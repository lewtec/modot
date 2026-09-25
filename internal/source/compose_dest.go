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
	"strings"
	"time"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/compose"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/filespine"
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
// profile. The system tree holds etc, usr, var, and bin as paths.
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
			profile = &profileFiles{recorded: map[lewpath.Path]recordedFile{}}
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
		var recorded map[lewpath.Path]recordedFile
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

func applyFiles(ctx context.Context, tree *compose.Tree, base fs.FS, recorded map[lewpath.Path]recordedFile, targetBase string) ([]File, error) {
	var out []File
	for name, declared := range tree.All() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		label := strings.Join(name.Parts(), "/")
		sourceInfo, module := recordedNote(recorded, name)
		basic := BasicFile{
			RelPathStr:    label,
			TargetBaseDir: targetBase,
			FileMode:      permission(declared.Mode),
			Info:          sourceInfo,
			Module:        module,
		}
		if declared.Type == compose.TypeLink {
			target, ok := linkTarget(declared)
			if !ok {
				return nil, fmt.Errorf("file %s: %w", label, compose.ErrSlot)
			}
			basic.FileType = TypeSymlink
			absolute := ""
			if found, exists := recorded[name]; exists {
				absolute = found.absolutePath
			}
			out = append(out, &StaticFile{BasicFile: basic, AbsPath: absolute, Link: target})
			continue
		}
		if found, ok := recorded[name]; ok && found.absolutePath != "" && declared.Type == compose.TypeRef {
			basic.FileType = TypeStatic
			out = append(out, &StaticFile{BasicFile: basic, AbsPath: found.absolutePath})
			continue
		}
		body, err := compose.Encode(declared, base)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", label, err)
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
	var nested lewpath.Path
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
	if nested == lewpath.New(".") {
		return chosen, file, nil
	}
	combined := nested.Join(file.RelPath())
	return chosen, rebasedFile{File: file, rel: strings.Join(combined.Parts(), "/")}, nil
}

func insideDir(root, target string) (lewpath.Path, bool) {
	if root == "" || target == "" {
		return lewpath.Path{}, false
	}
	rel, err := lewpath.New(target).Rel(lewpath.New(root))
	if err != nil {
		return lewpath.Path{}, false
	}
	parts := rel.Parts()
	if len(parts) > 0 && parts[0] == ".." {
		return lewpath.Path{}, false
	}
	return rel, true
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
		if current == lewpath.New(".") {
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
			return addLink(tree, fsys, current, name)
		}
		return addRef(tree, current, name, entry)
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
	entries, err := directory.ReadDir(fsys)
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
		body, err := child.ReadFile(fsys)
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

func addLink(tree *compose.Tree, fsys fs.FS, name lewpath.Path, raw string) error {
	target, err := fs.ReadLink(fsys, raw)
	if err != nil {
		return err
	}
	return tree.Add(name, compose.File{
		Type:   compose.TypeLink,
		Mode:   lstatPerm(fsys, name),
		Values: map[string]compose.Slot{"target": compose.Link(target)},
	})
}

func addRef(tree *compose.Tree, name lewpath.Path, raw string, entry fs.DirEntry) error {
	return tree.Add(name, compose.File{
		Type:   compose.TypeRef,
		Mode:   entryPerm(entry),
		Values: map[string]compose.Slot{"src": compose.Ref(raw)},
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

func recordedNote(recorded map[lewpath.Path]recordedFile, name lewpath.Path) (string, string) {
	if found, ok := recorded[name]; ok {
		return noteText(found, strings.Join(name.Parts(), "/"))
	}
	directory := name.Parent().Join(name.Name() + ".d.tmpl")
	var best lewpath.Path
	var chosen recordedFile
	var foundChild bool
	for child := range directory.Under(recordedPaths(recorded)) {
		if child == directory {
			continue
		}
		if !foundChild || pathBefore(child, best) {
			best = child
			chosen = recorded[child]
			foundChild = true
		}
	}
	label := strings.Join(name.Parts(), "/")
	if !foundChild {
		return "filespine:" + label, ""
	}
	return noteText(chosen, label)
}

func pathBefore(left, right lewpath.Path) bool {
	a, b := left.Parts(), right.Parts()
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := range n {
		if a[i] == b[i] {
			continue
		}
		return a[i] < b[i]
	}
	return len(a) < len(b)
}

func noteText(recorded recordedFile, name string) (string, string) {
	if recorded.sourceInfo == "" {
		return "filespine:" + name, recorded.module
	}
	return recorded.sourceInfo, recorded.module
}

func recordedPaths(recorded map[lewpath.Path]recordedFile) iter.Seq2[lewpath.Path, error] {
	return func(yield func(lewpath.Path, error) bool) {
		for pathName := range recorded {
			if !yield(pathName, nil) {
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
	recorded map[lewpath.Path]recordedFile
	direct   map[string]File
	paths    pathSet
}

func indexTree(tree *compose.Tree) pathSet {
	var set pathSet
	if tree == nil {
		return set
	}
	for name := range tree.All() {
		set.add(name)
	}
	return set
}

func needsMerge(rel string, cue pathSet) bool {
	if strings.Contains(rel, ".d.tmpl/") || strings.HasSuffix(rel, ".d.tmpl") {
		return true
	}
	return cue.overlaps(rel)
}

func linesDest(rel lewpath.Path) (lewpath.Path, bool) {
	for current := rel.Parent(); current != lewpath.New("."); current = current.Parent() {
		if dotDirectory(current) {
			dest, err := linesPath(current)
			if err != nil {
				return lewpath.Path{}, false
			}
			return dest, true
		}
		if current.Parent() == current {
			break
		}
	}
	return lewpath.Path{}, false
}

func directFile(direct map[string]File, name lewpath.Path) (File, bool) {
	for rel, file := range direct {
		if lewpath.New(rel) == name {
			return file, true
		}
	}
	return nil, false
}

type pathSet struct {
	files map[lewpath.Path]struct{}
	dirs  map[lewpath.Path]struct{}
}

func (set *pathSet) add(name lewpath.Path) {
	if set.files == nil {
		set.files = map[lewpath.Path]struct{}{}
		set.dirs = map[lewpath.Path]struct{}{}
	}
	set.files[name] = struct{}{}
	for parent := name.Parent(); parent != lewpath.New("."); parent = parent.Parent() {
		set.dirs[parent] = struct{}{}
		if parent.Parent() == parent {
			break
		}
	}
}

func (set *pathSet) overlaps(path string) bool {
	if set == nil || set.files == nil {
		return false
	}
	name := lewpath.New(path)
	if _, ok := set.files[name]; ok {
		return true
	}
	if _, ok := set.dirs[name]; ok {
		return true
	}
	for parent := name.Parent(); parent != lewpath.New("."); parent = parent.Parent() {
		if _, ok := set.files[parent]; ok {
			return true
		}
		if parent.Parent() == parent {
			break
		}
	}
	return false
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
	profile.paths.add(lewpath.New(rel))
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
		file, exists := directFile(profile.direct, dest)
		if !exists {
			continue
		}
		for key := range profile.direct {
			if lewpath.New(key) == dest {
				delete(profile.direct, key)
				break
			}
		}
		if err := profile.add(file); err != nil {
			return err
		}
	}
	return nil
}

func (profile *profileFiles) add(file File) error {
	rel := file.RelPath()
	name := lewpath.New(rel)
	if !name.Valid() || name.IsAbs() || name == lewpath.New(".") {
		return fmt.Errorf("file %s: %w", rel, errInvalidRelativePath)
	}
	recorded, member, err := fileMember(name, file)
	if err != nil {
		return err
	}
	if previous, ok := profile.recorded[name]; ok {
		if !previous.same(recorded) {
			return fmt.Errorf("file %s: %w", rel, errPathConflict)
		}
		return nil
	}
	profile.recorded[name] = recorded
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
			links[strings.Join(name.Parts(), "/")] = recorded.linkTarget
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
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
		}
		body, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
		}
		if closeErr != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), closeErr)
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
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), errStaticNoSource)
	}
	abs := lewpath.New(staticFile.AbsPath)
	root, err := filespine.OpenDir(abs.Parent())
	if err != nil {
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
	}
	defer root.Close()
	base := lewpath.New(abs.Name())
	linkInfo, err := base.Lstat(root)
	if err != nil {
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
	}
	recorded.absolutePath = staticFile.AbsPath
	mode = permission(linkInfo.Mode())
	if linkInfo.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Readlink(staticFile.AbsPath)
		if err != nil {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
		}
		recorded.linkTarget = target
		return recorded, lewfs.File{Name: name, Mode: mode | fs.ModeSymlink, Reader: bytes.NewReader(nil)}, nil
	}
	opened, err := base.Open(root)
	if err != nil {
		if recorded.linkTarget == "" {
			return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
		}
		return recorded, lewfs.File{Name: name, Mode: mode, Reader: bytes.NewReader(nil)}, nil
	}
	openedInfo, err := base.Stat(root)
	if err != nil {
		opened.Close()
		return recordedFile{}, lewfs.File{}, fmt.Errorf("file %s: %w", file.RelPath(), err)
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
