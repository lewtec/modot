package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/compose"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
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
	var built *Tree
	err := taskgroup.GoIsolated(ctx, "compose", taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
		var err error
		built, err = composeTracked(ctx, status, request)
		return err
	})
	return built, err
}

func composeTracked(ctx context.Context, status *taskgroup.Status, request destRequest) (*Tree, error) {
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

	cuePaths := map[string]pathSet{}
	for _, name := range filespine.Visible(mode) {
		cuePaths[name] = indexTree(profiles[name])
	}
	total := int64(len(request.files))
	if status != nil {
		status.Progress(0, total)
	}
	grouped := map[string]*profileFiles{}
	for i, file := range request.files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if status != nil && (i%256 == 0 || int64(i) == total-1) {
			status.Progress(int64(i), total)
		}
		name, placed, err := placeInProfile(file, mode, request.targetBase)
		if err != nil {
			return nil, err
		}
		profile := grouped[name]
		if profile == nil {
			profile = &profileFiles{recorded: map[string]recordedFile{}}
			grouped[name] = profile
		}
		// Plain copies do not enter Tree.Add. That scan is quadratic, and a
		// bundle fingerprint in SourceInfo is what lets plan skip per-file hashes.
		if needsMerge(placed.RelPath(), cuePaths[name]) {
			if err := profile.add(placed); err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			continue
		}
		if err := profile.addDirect(placed); err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
	}
	if status != nil {
		status.Progress(total, total)
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
			if err := profile.absorbLineTargets(); err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			base, err = profile.filesystem(ctx)
			if err != nil {
				return nil, fmt.Errorf("file.%s: %w", name, err)
			}
			recorded = profile.recorded
			squashed, err := squashContext(ctx, base)
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
		applied, err := applyFiles(ctx, tree, base, recorded, filespine.ApplyDir(name, request.targetBase))
		if err != nil {
			return nil, fmt.Errorf("file.%s: %w", name, err)
		}
		apply = append(apply, applied...)
		for _, file := range profileDirect(profile, filespine.ApplyDir(name, request.targetBase)) {
			apply = append(apply, file)
		}
	}
	return &Tree{dest: profileFilesystem{filesystems: filesystems, direct: directOpen(grouped, mode, request.targetBase)}, targetBase: request.targetBase, files: apply}, nil
}

func profileDirect(profile *profileFiles, base string) []File {
	if profile == nil {
		return nil
	}
	out := make([]File, 0, len(profile.direct))
	for rel, file := range profile.direct {
		if file.RelPath() == rel && file.TargetBase() == base {
			out = append(out, file)
			continue
		}
		out = append(out, rootedFile{File: file, rel: rel, base: base})
	}
	return out
}

func directOpen(grouped map[string]*profileFiles, mode, primary string) map[string]File {
	out := map[string]File{}
	for _, name := range filespine.Visible(mode) {
		profile := grouped[name]
		if profile == nil {
			continue
		}
		for rel, file := range profile.direct {
			out[rel] = rootedFile{File: file, rel: rel, base: filespine.ApplyDir(name, primary)}
		}
	}
	return out
}

// profileFilesystem opens the first profile filesystem that has name.
// Two profiles may use the same relative path with different apply directories.
// Files keeps both. Open returns the first match.
type profileFilesystem struct {
	filesystems []fs.FS
	direct      map[string]File
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
	if file, ok := filesystem.direct[name]; ok {
		reader, err := file.Reader()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		info := sourceInfo{name: lewpath.New(name).Name(), mode: permission(file.Mode()), size: int64(len(body))}
		return &sourceFile{info: info, Reader: bytes.NewReader(body)}, nil
	}
	if missing == nil {
		missing = &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return nil, missing
}

func applyFiles(ctx context.Context, tree *compose.Tree, base fs.FS, recorded map[string]recordedFile, targetBase string) ([]File, error) {
	var out []File
	for name, declared := range tree.All() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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

// placeInProfile picks the visible profile whose directory contains the file target.
// A target nested under that directory keeps the extra path on the relative name.
func placeInProfile(file File, mode, primary string) (string, File, error) {
	target := file.TargetBase()
	if target == "" {
		target = primary
	}
	var chosen string
	var chosenDir string
	var nested string
	for _, name := range filespine.Visible(mode) {
		dir := filespine.ApplyDir(name, primary)
		sub, ok := insideDir(dir, target)
		if !ok {
			continue
		}
		if chosen == "" || len(dir) > len(chosenDir) {
			chosen = name
			chosenDir = dir
			nested = sub
		}
	}
	if chosen == "" {
		return "", nil, fmt.Errorf("file %s: target %s: %w", file.RelPath(), file.TargetBase(), errNotProfileDir)
	}
	if nested == "" {
		return chosen, file, nil
	}
	return chosen, rebasedFile{File: file, rel: lewpath.New(nested, file.RelPath()).String()}, nil
}

func insideDir(root, target string) (string, bool) {
	if root == "" || target == "" {
		return "", false
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	if rel == "." {
		return "", true
	}
	return filepath.ToSlash(rel), true
}

type rebasedFile struct {
	File
	rel string
}

func (file rebasedFile) RelPath() string { return file.rel }

// squashContext is compose.Squash that returns when ctx is cancelled.
// The library walk does not look at a context, so a large tree would
// keep merging after the session stops.
func squashContext(ctx context.Context, fsys fs.FS) (*compose.Tree, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if fsys == nil {
		return compose.New(), nil
	}
	tree := compose.New()
	err := lewpath.New(".").WalkDir(fsys, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		current := lewpath.New(name)
		if current.String() == "." {
			return nil
		}
		if dotDirectory(current) {
			if !entry.IsDir() {
				return fmt.Errorf("file %s: %w", current, compose.ErrPath)
			}
			if err := addDotDirectory(ctx, tree, fsys, current); err != nil {
				return err
			}
			return fs.SkipDir
		}
		if entry.IsDir() || current.Suffix() == ".tmpl" {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return addLink(tree, fsys, current)
		}
		return addRef(tree, current, entry)
	})
	if err != nil {
		return nil, err
	}
	return tree, nil
}

func addDotDirectory(ctx context.Context, tree *compose.Tree, fsys fs.FS, directory lewpath.Path) error {
	destination, err := linesPath(directory)
	if err != nil {
		return fmt.Errorf("file %s: %w", directory, err)
	}
	entries, err := fs.ReadDir(fsys, directory.String())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		child := directory.Join(entry.Name())
		if entry.IsDir() {
			return fmt.Errorf("file %s: %w", child, compose.ErrPath)
		}
		if child.Suffix() == ".tmpl" {
			continue
		}
		body, err := fs.ReadFile(fsys, child.String())
		if err != nil {
			return err
		}
		err = tree.Add(destination, compose.File{
			Type:   compose.TypeLines,
			Values: map[string]compose.Slot{child.Name(): compose.Text(string(body))},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func dotDirectory(name lewpath.Path) bool {
	suffixes := name.Suffixes()
	count := len(suffixes)
	return count >= 2 && suffixes[count-2] == ".d" && suffixes[count-1] == ".tmpl"
}

func linesPath(directory lewpath.Path) (lewpath.Path, error) {
	if !dotDirectory(directory) {
		return lewpath.Path{}, compose.ErrPath
	}
	trimmed := directory.WithSuffix("").WithSuffix("")
	base := trimmed.Name()
	if base == "" || base == "." || base == ".." {
		return lewpath.Path{}, compose.ErrPath
	}
	return trimmed, nil
}

func addLink(tree *compose.Tree, fsys fs.FS, name lewpath.Path) error {
	target, err := name.ReadLink(fsys)
	if err != nil {
		return err
	}
	return tree.Add(name, compose.File{
		Type:   compose.TypeLink,
		Mode:   lstatPerm(fsys, name),
		Values: map[string]compose.Slot{"target": compose.Link(target.String())},
	})
}

func addRef(tree *compose.Tree, name lewpath.Path, entry fs.DirEntry) error {
	return tree.Add(name, compose.File{
		Type:   compose.TypeRef,
		Mode:   entryPerm(entry),
		Values: map[string]compose.Slot{"src": compose.Ref(name.String())},
	})
}

func lstatPerm(fsys fs.FS, name lewpath.Path) fs.FileMode {
	info, err := name.Lstat(fsys)
	if err != nil {
		return 0o644
	}
	return permission(info.Mode())
}

func entryPerm(entry fs.DirEntry) fs.FileMode {
	info, err := entry.Info()
	if err != nil {
		return 0o644
	}
	return permission(info.Mode())
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
	direct   map[string]File
	paths    pathSet
}

func indexTree(tree *compose.Tree) pathSet {
	var set pathSet
	if tree == nil {
		return set
	}
	for name := range tree.All() {
		set.add(name.String())
	}
	return set
}

func needsMerge(rel string, cue pathSet) bool {
	if strings.Contains(rel, ".d.tmpl/") || strings.HasSuffix(rel, ".d.tmpl") {
		return true
	}
	return cue.overlaps(rel)
}

func linesDest(rel string) (string, bool) {
	const marker = ".d.tmpl"
	index := strings.Index(rel, marker)
	if index <= 0 {
		return "", false
	}
	return rel[:index], true
}

type pathSet struct {
	files map[string]struct{}
	dirs  map[string]struct{}
}

func (set *pathSet) add(path string) {
	if set.files == nil {
		set.files = map[string]struct{}{}
		set.dirs = map[string]struct{}{}
	}
	set.files[path] = struct{}{}
	for parent := pathParent(path); parent != ""; parent = pathParent(parent) {
		set.dirs[parent] = struct{}{}
	}
}

func (set *pathSet) overlaps(path string) bool {
	if set == nil || set.files == nil {
		return false
	}
	if _, ok := set.files[path]; ok {
		return true
	}
	if _, ok := set.dirs[path]; ok {
		return true
	}
	for parent := pathParent(path); parent != ""; parent = pathParent(parent) {
		if _, ok := set.files[parent]; ok {
			return true
		}
	}
	return false
}

func pathParent(path string) string {
	index := strings.LastIndex(path, "/")
	if index <= 0 {
		return ""
	}
	return path[:index]
}

type rootedFile struct {
	File
	rel  string
	base string
}

func (file rootedFile) RelPath() string    { return file.rel }
func (file rootedFile) TargetBase() string { return file.base }

func (profile *profileFiles) addDirect(file File) error {
	rel := file.RelPath()
	if profile.direct == nil {
		profile.direct = map[string]File{}
	}
	if _, ok := profile.direct[rel]; ok {
		return fmt.Errorf("file %s: %w", rel, errPathConflict)
	}
	if profile.paths.overlaps(rel) {
		return fmt.Errorf("file %s: %w", rel, compose.ErrPath)
	}
	profile.direct[rel] = markSymlink(file)
	profile.paths.add(rel)
	return nil
}

func markSymlink(file File) File {
	static, ok := unwrapStatic(file)
	if !ok || static.AbsPath == "" || static.Type() == TypeSymlink {
		return file
	}
	info, err := os.Lstat(static.AbsPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return file
	}
	target, err := os.Readlink(static.AbsPath)
	if err != nil {
		return file
	}
	clone := *static
	clone.FileType = TypeSymlink
	clone.Link = target
	return &clone
}

func unwrapStatic(file File) (*StaticFile, bool) {
	switch file := file.(type) {
	case *StaticFile:
		return file, true
	case rebasedFile:
		return unwrapStatic(file.File)
	case rootedFile:
		return unwrapStatic(file.File)
	default:
		return nil, false
	}
}

func (profile *profileFiles) absorbLineTargets() error {
	for rel := range profile.recorded {
		dest, ok := linesDest(rel)
		if !ok {
			continue
		}
		file, exists := profile.direct[dest]
		if !exists {
			continue
		}
		delete(profile.direct, dest)
		if err := profile.add(file); err != nil {
			return err
		}
	}
	return nil
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

type sourceFile struct {
	info fs.FileInfo
	*bytes.Reader
}

func (file *sourceFile) Stat() (fs.FileInfo, error) { return file.info, nil }

func (file *sourceFile) Close() error { return nil }

type sourceInfo struct {
	name string
	mode fs.FileMode
	size int64
}

func (info sourceInfo) Name() string       { return info.name }
func (info sourceInfo) Size() int64        { return info.size }
func (info sourceInfo) Mode() fs.FileMode  { return info.mode }
func (info sourceInfo) ModTime() time.Time { return time.Time{} }
func (info sourceInfo) IsDir() bool        { return info.mode.IsDir() }
func (info sourceInfo) Sys() any           { return nil }

func permission(mode fs.FileMode) fs.FileMode {
	mode = mode.Perm()
	if mode == 0 {
		return 0o644
	}
	return mode
}
