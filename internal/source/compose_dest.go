package source

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/fs/compose"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/pkg/filespine"
)

var (
	errInvalidRelPath = errors.New("invalid path")
	errStaticNoSource = errors.New("static file has no source path")
	errPathConflict   = errors.New("conflict")
	errPathClash      = errors.New("path clash")
	errNotProfileDir  = errors.New("target is not a profile directory")
)

// composeApply merges visible CUE profiles with discovered files.
// Each profile is one compose tree. The apply target receives the primary
// profile. Fixed profiles keep NamespaceBase.
func composeApply(cfg *configcue.Config, targetBase string, discovered []File) (*Tree, error) {
	mode := filespine.ModeHome
	var profiles map[string]*compose.Tree
	var err error
	if cfg != nil {
		mode = cfg.RuntimeMode()
		profiles, err = cfg.FileProfiles()
		if err != nil {
			return nil, err
		}
	}
	if profiles == nil {
		profiles = map[string]*compose.Tree{}
		for _, name := range filespine.Profiles {
			if filespine.NamespaceVisible(mode, name) {
				profiles[name] = compose.New()
			}
		}
	}

	grouped := map[string]*sourceFS{}
	for _, f := range discovered {
		name, err := profileFor(f, mode, targetBase)
		if err != nil {
			return nil, err
		}
		src := grouped[name]
		if src == nil {
			src = newSourceFS()
			grouped[name] = src
		}
		if err := src.add(f); err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
	}

	var apply []File
	var views []fs.FS
	for _, name := range filespine.Profiles {
		tree := profiles[name]
		if tree == nil {
			continue
		}
		src := grouped[name]
		if src != nil {
			squashed, err := compose.Squash(src)
			if err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			if err := tree.Merge(squashed); err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
		}
		var base fs.FS
		if src != nil {
			base = src
		}
		fsys, err := tree.FS(base)
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		views = append(views, fsys)
		got, err := filesFrom(fsys, src, profileBase(name, targetBase))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		apply = append(apply, got...)
	}
	return &Tree{dest: multiFS{views: views}, targetBase: targetBase, files: apply}, nil
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

func profileFor(f File, mode, primaryBase string) (string, error) {
	base := filepath.Clean(f.TargetBase())
	primary := filepath.Clean(primaryBase)
	if f.TargetBase() == "" || base == primary {
		return filespine.Primary(mode), nil
	}
	for _, name := range filespine.Profiles {
		fixed, ok := filespine.NamespaceBase[name]
		if !ok || !filespine.NamespaceVisible(mode, name) {
			continue
		}
		if filepath.Clean(fixed) == base {
			return name, nil
		}
	}
	return "", fmt.Errorf("file %s: target %s: %w", f.RelPath(), f.TargetBase(), errNotProfileDir)
}

func profileBase(name, primaryBase string) string {
	if base, ok := filespine.NamespaceBase[name]; ok {
		return base
	}
	return primaryBase
}

func filesFrom(fsys fs.FS, src *sourceFS, targetBase string) ([]File, error) {
	var out []File
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, walkErr error) error {
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
		originInfo, module := originOf(src, name)
		rel := filepath.FromSlash(name)
		if src != nil {
			if node, ok := src.node(name); ok && node.link != "" {
				out = append(out, &StaticFile{
					BasicFile: BasicFile{
						RelPathStr:    rel,
						TargetBaseDir: targetBase,
						FileMode:      mode,
						Info:          originInfo,
						FileType:      TypeSymlink,
						Module:        module,
					},
					AbsPath: node.abs,
				})
				return nil
			}
			if node, ok := src.node(name); ok && node.abs != "" {
				out = append(out, &StaticFile{
					BasicFile: BasicFile{
						RelPathStr:    rel,
						TargetBaseDir: targetBase,
						FileMode:      mode,
						Info:          originInfo,
						FileType:      TypeStatic,
						Module:        module,
					},
					AbsPath: node.abs,
				})
				return nil
			}
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		out = append(out, &BufferFile{
			BasicFile: BasicFile{
				RelPathStr:    rel,
				TargetBaseDir: targetBase,
				FileMode:      mode,
				Info:          originInfo,
				FileType:      TypeStatic,
				Module:        module,
			},
			Content: body,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func originOf(src *sourceFS, name string) (string, string) {
	if src != nil {
		if info, module, ok := src.meta(name); ok {
			if info == "" {
				info = "filespine:" + name
			}
			return info, module
		}
	}
	return "filespine:" + name, ""
}

type sourceNode struct {
	body   []byte
	abs    string
	link   string
	mode   fs.FileMode
	info   string
	module string
}

type sourceFS struct {
	files map[string]sourceNode
}

func newSourceFS() *sourceFS {
	return &sourceFS{files: map[string]sourceNode{}}
}

func (s *sourceFS) add(f File) error {
	name := path.Clean(filepath.ToSlash(f.RelPath()))
	if !fs.ValidPath(name) || name == "." {
		return fmt.Errorf("file %s: %w", f.RelPath(), errInvalidRelPath)
	}
	node := sourceNode{
		mode:   f.Mode().Perm(),
		info:   f.SourceInfo(),
		module: moduleNameOf(f),
	}
	if node.mode == 0 {
		node.mode = 0o644
	}
	if sf, ok := f.(*StaticFile); ok {
		if sf.AbsPath == "" {
			return fmt.Errorf("file %s: %w", name, errStaticNoSource)
		}
		st, err := os.Lstat(sf.AbsPath)
		if err != nil {
			return fmt.Errorf("file %s: %w", name, err)
		}
		node.abs = sf.AbsPath
		node.mode = st.Mode().Perm()
		if node.mode == 0 {
			node.mode = 0o644
		}
		if st.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(sf.AbsPath)
			if err != nil {
				return fmt.Errorf("file %s: %w", name, err)
			}
			node.link = link
		}
		return s.put(name, node)
	}
	reader, err := f.Reader()
	if err != nil {
		return fmt.Errorf("file %s: %w", name, err)
	}
	body, err := io.ReadAll(reader)
	closeErr := reader.Close()
	if err != nil {
		return fmt.Errorf("file %s: %w", name, err)
	}
	if closeErr != nil {
		return fmt.Errorf("file %s: %w", name, closeErr)
	}
	node.body = body
	return s.put(name, node)
}

func (s *sourceFS) put(name string, node sourceNode) error {
	for existing := range s.files {
		if existing == name {
			prev := s.files[existing]
			if prev.abs != node.abs || prev.link != node.link || !bytes.Equal(prev.body, node.body) {
				return fmt.Errorf("file %s: %w", name, errPathConflict)
			}
			return nil
		}
		if strings.HasPrefix(existing, name+"/") || strings.HasPrefix(name, existing+"/") {
			return fmt.Errorf("file %s: path clash with %s: %w", name, existing, errPathClash)
		}
	}
	if s.files == nil {
		s.files = map[string]sourceNode{}
	}
	s.files[name] = node
	return nil
}

func (s *sourceFS) node(name string) (sourceNode, bool) {
	if s == nil {
		return sourceNode{}, false
	}
	node, ok := s.files[name]
	return node, ok
}

func (s *sourceFS) meta(name string) (string, string, bool) {
	if s == nil {
		return "", "", false
	}
	if node, ok := s.files[name]; ok {
		return node.info, node.module, true
	}
	prefix := name + ".d.tmpl/"
	var bestPath string
	var best sourceNode
	for pathName, node := range s.files {
		if strings.HasPrefix(pathName, prefix) && (bestPath == "" || pathName < bestPath) {
			bestPath = pathName
			best = node
		}
	}
	if bestPath == "" {
		return "", "", false
	}
	return best.info, best.module, true
}

func (s *sourceFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if s.isDir(name) {
		return &sourceDir{name: name, entries: s.children(name)}, nil
	}
	node, ok := s.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if node.abs != "" {
		opened, err := os.Open(node.abs)
		if err != nil {
			return nil, err
		}
		return opened, nil
	}
	return &sourceFile{
		info:   sourceInfo{name: path.Base(name), mode: fileTypeMode(node), size: int64(len(node.body))},
		Reader: bytes.NewReader(node.body),
	}, nil
}

func (s *sourceFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if !s.isDir(name) {
		if _, ok := s.files[name]; ok {
			return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
		}
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	return s.children(name), nil
}

func (s *sourceFS) isDir(name string) bool {
	if name == "." {
		return true
	}
	prefix := name + "/"
	for pathName := range s.files {
		if strings.HasPrefix(pathName, prefix) {
			return true
		}
	}
	return false
}

func (s *sourceFS) children(name string) []fs.DirEntry {
	prefix := ""
	if name != "." {
		prefix = name + "/"
	}
	type child struct {
		dir  bool
		node sourceNode
	}
	kids := map[string]child{}
	for pathName, node := range s.files {
		rest := pathName
		if prefix != "" {
			if !strings.HasPrefix(pathName, prefix) {
				continue
			}
			rest = strings.TrimPrefix(pathName, prefix)
		}
		elem, _, found := strings.Cut(rest, "/")
		if elem == "" {
			continue
		}
		if found {
			kids[elem] = child{dir: true}
			continue
		}
		if prev, ok := kids[elem]; ok && prev.dir {
			continue
		}
		kids[elem] = child{node: node}
	}
	names := make([]string, 0, len(kids))
	for name := range kids {
		names = append(names, name)
	}
	slices.Sort(names)
	out := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		item := kids[name]
		if item.dir {
			out = append(out, sourceEntry{name: name, mode: fs.ModeDir | 0o755})
			continue
		}
		out = append(out, sourceEntry{
			name: name,
			mode: fileTypeMode(item.node),
			size: int64(len(item.node.body)),
		})
	}
	return out
}

func fileTypeMode(node sourceNode) fs.FileMode {
	mode := node.mode.Perm()
	if mode == 0 {
		mode = 0o644
	}
	if node.link != "" {
		mode |= fs.ModeSymlink
	}
	return mode
}

type sourceEntry struct {
	name string
	mode fs.FileMode
	size int64
}

func (e sourceEntry) Name() string      { return e.name }
func (e sourceEntry) IsDir() bool       { return e.mode.IsDir() }
func (e sourceEntry) Type() fs.FileMode { return e.mode.Type() }
func (e sourceEntry) Info() (fs.FileInfo, error) {
	return sourceInfo{name: e.name, mode: e.mode, size: e.size}, nil
}

type sourceInfo struct {
	name string
	mode fs.FileMode
	size int64
}

func (i sourceInfo) Name() string       { return i.name }
func (i sourceInfo) Size() int64        { return i.size }
func (i sourceInfo) Mode() fs.FileMode  { return i.mode }
func (i sourceInfo) ModTime() time.Time { return time.Time{} }
func (i sourceInfo) IsDir() bool        { return i.mode.IsDir() }
func (i sourceInfo) Sys() any           { return nil }

type sourceFile struct {
	info fs.FileInfo
	*bytes.Reader
}

func (f *sourceFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *sourceFile) Close() error { return nil }

type sourceDir struct {
	name    string
	entries []fs.DirEntry
}

func (d *sourceDir) Stat() (fs.FileInfo, error) {
	return sourceInfo{name: path.Base(d.name), mode: fs.ModeDir | 0o755}, nil
}

func (d *sourceDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: fs.ErrInvalid}
}

func (d *sourceDir) Close() error { return nil }

func (d *sourceDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n == 0 {
		return nil, nil
	}
	if len(d.entries) == 0 {
		if n > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}
	if n < 0 || n > len(d.entries) {
		out := d.entries
		d.entries = nil
		return out, nil
	}
	out := d.entries[:n]
	d.entries = d.entries[n:]
	return out, nil
}
