package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/compose"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/pkg/filespine"
)

var (
	errInvalidRelativePath = errors.New("invalid path")
	errStaticNoSource      = errors.New("static file has no source path")
	errPathConflict        = errors.New("conflict")
	errNotProfileDir       = errors.New("target is not a profile directory")
)

// destRequest is one compose of cue profiles and discovered files.
type destRequest struct {
	config     *configcue.Config
	targetBase string
	files      []File
}

// composeApply merges visible CUE profiles with discovered files.
// Each profile is one compose tree. The apply target receives the primary
// profile. Fixed profiles keep NamespaceBase.
func composeApply(ctx context.Context, request destRequest) (*Tree, error) {
	mode := filespine.ModeHome
	var profiles map[string]*compose.Tree
	var err error
	if request.config != nil {
		mode = request.config.RuntimeMode()
		profiles, err = request.config.FileProfiles()
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

	grouped := map[string]*profileFiles{}
	for _, file := range request.files {
		name, ok := filespine.ProfileForTarget(mode, file.TargetBase(), request.targetBase)
		if !ok {
			return nil, fmt.Errorf("file %s: target %s: %w", file.RelPath(), file.TargetBase(), errNotProfileDir)
		}
		profile := grouped[name]
		if profile == nil {
			profile = &profileFiles{recorded: map[string]recordedFile{}}
			grouped[name] = profile
		}
		if err := profile.add(file); err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
	}

	var apply []File
	var filesystems []fs.FS
	for _, name := range filespine.Visible(mode) {
		tree := profiles[name]
		if tree == nil {
			continue
		}
		profile := grouped[name]
		var base fs.FS
		var recorded map[string]recordedFile
		if profile != nil {
			base, err = profile.filesystem(ctx)
			if err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			recorded = profile.recorded
			squashed, err := compose.Squash(base)
			if err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			if err := tree.Merge(squashed); err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
		}
		filesystem, err := tree.FS(base)
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		filesystems = append(filesystems, filesystem)
		applied, err := applyFiles(tree, base, recorded, filespine.ApplyDir(name, request.targetBase))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		apply = append(apply, applied...)
	}
	return &Tree{dest: profileFilesystem{filesystems: filesystems}, targetBase: request.targetBase, files: apply}, nil
}

// profileFilesystem opens the first profile filesystem that has name.
// Two profiles may use the same relative path with different apply directories.
// Files keeps both. Open returns the first match.
type profileFilesystem struct {
	filesystems []fs.FS
}

func (filesystem profileFilesystem) Open(name string) (fs.File, error) {
	var missing error
	for _, view := range filesystem.filesystems {
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

func applyFiles(tree *compose.Tree, base fs.FS, recorded map[string]recordedFile, targetBase string) ([]File, error) {
	var out []File
	for name, declared := range tree.All() {
		key := name.String()
		sourceInfo, module := recordedNote(recorded, key)
		basic := BasicFile{
			RelPathStr:    key,
			TargetBaseDir: targetBase,
			FileMode:      permission(declared.Mode),
			Info:          sourceInfo,
			Module:        module,
		}
		if declared.Type == compose.TypeLink {
			target, ok := linkTarget(declared)
			if !ok {
				return nil, fmt.Errorf("file %s: %w", key, compose.ErrSlot)
			}
			basic.FileType = TypeSymlink
			absolute := ""
			if found, exists := recorded[key]; exists {
				absolute = found.absolutePath
			}
			out = append(out, &StaticFile{BasicFile: basic, AbsPath: absolute, Link: target})
			continue
		}
		if found, ok := recorded[key]; ok && found.absolutePath != "" && declared.Type == compose.TypeRef {
			basic.FileType = TypeStatic
			out = append(out, &StaticFile{BasicFile: basic, AbsPath: found.absolutePath})
			continue
		}
		body, err := compose.Encode(declared, base)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", key, err)
		}
		basic.FileType = TypeStatic
		out = append(out, &BufferFile{BasicFile: basic, Content: body})
	}
	return out, nil
}

func linkTarget(file compose.File) (string, bool) {
	for _, slot := range file.Values {
		if target, ok := slot.Link(); ok {
			return target, true
		}
	}
	return "", false
}

func recordedNote(recorded map[string]recordedFile, name string) (string, string) {
	if found, ok := recorded[name]; ok {
		return noteText(found, name)
	}
	directory := lewpath.New(name + ".d.tmpl")
	var best string
	var chosen recordedFile
	for child := range directory.Under(recordedPaths(recorded)) {
		childName := child.String()
		if childName == directory.String() {
			continue
		}
		if best == "" || childName < best {
			best = childName
			chosen = recorded[childName]
		}
	}
	if best == "" {
		return "filespine:" + name, ""
	}
	return noteText(chosen, name)
}

func noteText(recorded recordedFile, name string) (string, string) {
	if recorded.sourceInfo == "" {
		return "filespine:" + name, recorded.module
	}
	return recorded.sourceInfo, recorded.module
}

func recordedPaths(recorded map[string]recordedFile) iter.Seq2[lewpath.Path, error] {
	return func(yield func(lewpath.Path, error) bool) {
		for pathName := range recorded {
			if !yield(lewpath.New(pathName), nil) {
				return
			}
		}
	}
}

type recordedFile struct {
	body         []byte
	absolutePath string
	linkTarget   string
	sourceInfo   string
	module       string
}

func (recorded recordedFile) same(other recordedFile) bool {
	return recorded.absolutePath == other.absolutePath &&
		recorded.linkTarget == other.linkTarget &&
		bytes.Equal(recorded.body, other.body)
}

type profileFiles struct {
	members  []lewfs.File
	recorded map[string]recordedFile
}

func (profile *profileFiles) add(file File) error {
	name := lewpath.New(file.RelPath())
	if !name.Valid() || name.IsAbs() || name.String() == "." {
		return fmt.Errorf("file %s: %w", file.RelPath(), errInvalidRelativePath)
	}
	key := name.String()
	recorded, member, err := fileMember(name, file)
	if err != nil {
		return err
	}
	if previous, ok := profile.recorded[key]; ok {
		if !previous.same(recorded) {
			return fmt.Errorf("file %s: %w", key, errPathConflict)
		}
		return nil
	}
	profile.recorded[key] = recorded
	profile.members = append(profile.members, member)
	return nil
}

func (profile *profileFiles) filesystem(ctx context.Context) (fs.FS, error) {
	base, err := lewfs.New(ctx, func(yield func(lewfs.File, error) bool) {
		for _, file := range profile.members {
			if !yield(file, nil) {
				return
			}
		}
	})
	if err != nil {
		return nil, err
	}
	links := map[string]string{}
	for name, recorded := range profile.recorded {
		if recorded.linkTarget != "" {
			links[name] = recorded.linkTarget
		}
	}
	if len(links) == 0 {
		return base, nil
	}
	return linkSource{FS: base, links: links}, nil
}

// linkSource is the source listing plus the symlink targets Squash reads.
type linkSource struct {
	fs.FS
	links map[string]string
}

func (source linkSource) ReadLink(name string) (string, error) {
	target, ok := source.links[name]
	if !ok {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrInvalid}
	}
	return target, nil
}

func (source linkSource) Lstat(name string) (fs.FileInfo, error) {
	return fs.Stat(source.FS, name)
}

func fileMember(name lewpath.Path, file File) (recordedFile, lewfs.File, error) {
	recorded := recordedFile{sourceInfo: file.SourceInfo(), module: moduleNameOf(file)}
	mode := permission(file.Mode())
	staticFile, ok := file.(*StaticFile)
	if !ok {
		reader, err := file.Reader()
		if err != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
		}
		body, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
		}
		if closeErr != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, closeErr)
		}
		recorded.body = body
		return recorded, lewfs.File{
			Name:   name,
			Mode:   mode,
			Size:   int64(len(body)),
			Reader: bytes.NewReader(body),
		}, nil
	}
	if staticFile.AbsPath == "" {
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, errStaticNoSource)
	}
	absolute := lewpath.New(staticFile.AbsPath)
	root, err := lewpath.Open(absolute.Parent().String())
	if err != nil {
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
	}
	defer root.Close()
	base := lewpath.New(absolute.Name())
	linkInfo, err := base.Lstat(root)
	if err != nil {
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
	}
	recorded.absolutePath = staticFile.AbsPath
	mode = permission(linkInfo.Mode())
	if linkInfo.Mode()&fs.ModeSymlink != 0 {
		target, err := base.ReadLink(root)
		if err != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
		}
		recorded.linkTarget = target.String()
		return recorded, lewfs.File{Name: name, Mode: mode | fs.ModeSymlink, Reader: bytes.NewReader(nil)}, nil
	}
	opened, err := base.Open(root)
	if err != nil {
		if recorded.linkTarget == "" {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
		}
		return recorded, lewfs.File{Name: name, Mode: mode, Reader: bytes.NewReader(nil)}, nil
	}
	openedInfo, err := base.Stat(root)
	if err != nil {
		opened.Close()
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", name, err)
	}
	return recorded, lewfs.File{Name: name, Mode: mode, Size: openedInfo.Size(), Reader: opened}, nil
}

func permission(mode fs.FileMode) fs.FileMode {
	mode = mode.Perm()
	if mode == 0 {
		return 0o644
	}
	return mode
}
