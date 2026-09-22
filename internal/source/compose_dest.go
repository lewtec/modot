package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/compose"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/pkg/filespine"
)

var (
	errInvalidRelPath = errors.New("invalid path")
	errStaticNoSource = errors.New("static file has no source path")
	errPathConflict   = errors.New("conflict")
	errNotProfileDir  = errors.New("target is not a profile directory")
)

// destRequest is one compose of cue profiles and discovered files.
type destRequest struct {
	cfg        *configcue.Config
	targetBase string
	files      []File
}

// composeApply merges visible CUE profiles with discovered files.
// Each profile is one compose tree. The apply target receives the primary
// profile. Fixed profiles keep NamespaceBase.
func composeApply(ctx context.Context, req destRequest) (*Tree, error) {
	mode := filespine.ModeHome
	var profiles map[string]*compose.Tree
	var err error
	if req.cfg != nil {
		mode = req.cfg.RuntimeMode()
		profiles, err = req.cfg.FileProfiles()
		if err != nil {
			return nil, err
		}
	}
	if len(profiles) == 0 {
		profiles = map[string]*compose.Tree{}
		for _, name := range filespine.Visible(mode) {
			profiles[name] = compose.New()
		}
	}

	grouped := map[string]*bucket{}
	for _, f := range req.files {
		name, ok := filespine.ProfileForTarget(mode, f.TargetBase(), req.targetBase)
		if !ok {
			return nil, fmt.Errorf("file %s: target %s: %w", f.RelPath(), f.TargetBase(), errNotProfileDir)
		}
		src := grouped[name]
		if src == nil {
			src = &bucket{origins: map[string]origin{}}
			grouped[name] = src
		}
		if err := src.add(f); err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
	}

	var apply []File
	var views []fs.FS
	for _, name := range filespine.Visible(mode) {
		tree := profiles[name]
		if tree == nil {
			continue
		}
		src := grouped[name]
		var base fs.FS
		var origins map[string]origin
		if src != nil {
			base, err = src.fs(ctx)
			if err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			origins = src.origins
			squashed, err := compose.Squash(base)
			if err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			if err := tree.Merge(squashed); err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
		}
		fsys, err := tree.FS(base)
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		views = append(views, fsys)
		got, err := filesFrom(fsys, origins, filespine.ApplyDir(name, req.targetBase))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		apply = append(apply, got...)
	}
	return &Tree{dest: multiFS{views: views}, targetBase: req.targetBase, files: apply}, nil
}

// multiFS opens the first profile filesystem that has name.
// Two profiles may use the same relative path with different apply directories.
// Files keeps both. Open returns the first match.
type multiFS struct {
	views []fs.FS
}

func (m multiFS) Open(name string) (fs.File, error) {
	var missing error
	for _, view := range m.views {
		if view == nil {
			continue
		}
		opened, err := view.Open(name)
		if err == nil {
			return opened, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			if missing == nil {
				missing = err
			}
			continue
		}
		return nil, err
	}
	if missing == nil {
		missing = &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return nil, missing
}

func filesFrom(fsys fs.FS, origins map[string]origin, targetBase string) ([]File, error) {
	var out []File
	err := lewpath.New(".").WalkDir(fsys, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		note, module := originNote(origins, name)
		rel := filepath.FromSlash(name)
		basic := BasicFile{
			RelPathStr:    rel,
			TargetBaseDir: targetBase,
			FileMode:      mode,
			Info:          note,
			Module:        module,
		}
		if src, ok := origins[name]; ok && src.link != "" {
			basic.FileType = TypeSymlink
			out = append(out, &StaticFile{BasicFile: basic, AbsPath: src.abs})
			return nil
		}
		if src, ok := origins[name]; ok && src.abs != "" {
			basic.FileType = TypeStatic
			out = append(out, &StaticFile{BasicFile: basic, AbsPath: src.abs})
			return nil
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		basic.FileType = TypeStatic
		out = append(out, &BufferFile{BasicFile: basic, Content: body})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func originNote(origins map[string]origin, name string) (string, string) {
	if src, ok := origins[name]; ok {
		if src.info == "" {
			return "filespine:" + name, src.module
		}
		return src.info, src.module
	}
	prefix := name + ".d.tmpl/"
	var bestPath string
	var best origin
	for pathName, src := range origins {
		if strings.HasPrefix(pathName, prefix) && (bestPath == "" || pathName < bestPath) {
			bestPath = pathName
			best = src
		}
	}
	if bestPath == "" {
		return "filespine:" + name, ""
	}
	if best.info == "" {
		return "filespine:" + name, best.module
	}
	return best.info, best.module
}

type origin struct {
	body   []byte
	abs    string
	link   string
	info   string
	module string
}

func (o origin) same(other origin) bool {
	return o.abs == other.abs && o.link == other.link && bytes.Equal(o.body, other.body)
}

type bucket struct {
	files   []lewfs.File
	origins map[string]origin
}

func (b *bucket) add(f File) error {
	name := lewpath.New(filepath.ToSlash(f.RelPath()))
	if !name.Valid() || name.IsAbs() || name.String() == "." {
		return fmt.Errorf("file %s: %w", f.RelPath(), errInvalidRelPath)
	}
	key := name.String()
	next, member, err := memberOf(name, f)
	if err != nil {
		return err
	}
	if prev, ok := b.origins[key]; ok {
		if !prev.same(next) {
			return fmt.Errorf("file %s: %w", key, errPathConflict)
		}
		return nil
	}
	b.origins[key] = next
	b.files = append(b.files, member)
	return nil
}

func (b *bucket) fs(ctx context.Context) (fs.FS, error) {
	return lewfs.New(ctx, func(yield func(lewfs.File, error) bool) {
		for _, file := range b.files {
			if !yield(file, nil) {
				return
			}
		}
	})
}

func memberOf(name lewpath.Path, f File) (origin, lewfs.File, error) {
	next := origin{info: f.SourceInfo(), module: moduleNameOf(f)}
	mode := f.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	if sf, ok := f.(*StaticFile); ok {
		if sf.AbsPath == "" {
			return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, errStaticNoSource)
		}
		st, err := os.Lstat(sf.AbsPath)
		if err != nil {
			return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
		}
		next.abs = sf.AbsPath
		mode = st.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		if st.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(sf.AbsPath)
			if err != nil {
				return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
			}
			next.link = link
		}
		opened, err := os.Open(sf.AbsPath)
		if err != nil {
			if next.link == "" {
				return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
			}
			return next, lewfs.File{Name: name, Mode: mode, Reader: bytes.NewReader(nil)}, nil
		}
		fst, err := opened.Stat()
		if err != nil {
			opened.Close()
			return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
		}
		return next, lewfs.File{Name: name, Mode: mode, Size: fst.Size(), Reader: opened}, nil
	}
	reader, err := f.Reader()
	if err != nil {
		return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
	}
	body, err := io.ReadAll(reader)
	closeErr := reader.Close()
	if err != nil {
		return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
	}
	if closeErr != nil {
		return origin{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, closeErr)
	}
	next.body = body
	return next, lewfs.File{
		Name:   name,
		Mode:   mode,
		Size:   int64(len(body)),
		Reader: bytes.NewReader(body),
	}, nil
}
