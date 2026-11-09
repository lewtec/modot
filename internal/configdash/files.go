package configdash

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/apply"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/filespine"
	"github.com/lewtec/modot/internal/source"
)

// FilesReport is the set of files a scope's apply would place, before it looks
// at the disk. Apply builds this tree from the same config the dashboard shows,
// so the paths here are what apply writes; whether each one changes or not is
// the deployer's opinion and needs the state file, which a preview does not read.
type FilesReport struct {
	// Mode is the runtime mode the tree was composed for (home, codebase,
	// system). It decides which file profiles are visible.
	Mode string
	// TargetBase is the root apply places files under on this host.
	TargetBase string
	// Profiles are the visible file profiles, in filespine order, e.g.
	// home and codebase for home mode.
	Profiles []string
	// Files is every entry in tree order. It is nil when Err is set.
	Files []MaterialFile
	// Warnings are the pipeline's non-fatal messages, e.g. a skipped source.
	Warnings []string
	// Err is set when the tree could not be built. The page still renders
	// what is known about the scope.
	Err error
}

// MaterialFile is one file apply places.
type MaterialFile struct {
	// Target is where the file lands, with the home directory abbreviated.
	Target string
	// Path is the file's path relative to its apply root.
	Path string
	// Dir is the directory part of Path; "" for a file at the root.
	Dir string
	// Name is the file's base name.
	Name string
	// Root is the apply root Target hangs from.
	Root string
	// Kind is the source kind: static, template, symlink, multifile, dotd.
	Kind string
	// Source is what produced the file: the module that ships it, the
	// file.<profile>.<dest> key of the place entry that mounted it, or the
	// pipeline stage for a copy.
	Source string
	// Mode is the permission bits, e.g. 0644.
	Mode string
	// Content is what apply would write for this file: the rendered bytes,
	// or why there are none (binary, too large).
	Content FileContent
}

// Text reports whether the file's content is available to show.
func (f MaterialFile) Text() bool { return f.Content.HasText() }

// ContentNote describes a body the page cannot show, empty when it can.
func (f MaterialFile) ContentNote() string { return contentErr(f.Content) }

// ContentLang is the language the body is written in, for the code block.
func (f MaterialFile) ContentLang() string { return contentLang(f.Path) }

// LoadFiles builds the file tree scope's apply would place from cwd. It renders
// templates and resolves modules the same way apply does, but writes nothing.
//
// A non-nil error always comes with a FilesReport carrying the scope's roots so
// the page can explain what failed next to them.
func LoadFiles(ctx context.Context, scope Scope, cwd string) (FilesReport, error) {
	opts := scope.DiscoverOptions(cwd)
	rep := FilesReport{
		Mode:       opts.RuntimeMode(),
		TargetBase: "",
		Profiles:   filespine.Visible(opts.RuntimeMode()),
		Files:      nil,
		Warnings:   nil,
		Err:        nil,
	}
	layers, err := Discover(ctx, scope, cwd)
	if err != nil {
		rep.Err = err

		return rep, err
	}

	result, err := configcue.Evaluate(ctx, opts)
	if err != nil {
		rep.Err = fmt.Errorf("evaluate %s layers: %w", scope, err)

		return rep, rep.Err
	}
	cfg, err := configcue.ConfigFromEvaluation(result)
	if err != nil {
		rep.Err = fmt.Errorf("decode evaluated config: %w", err)

		return rep, rep.Err
	}

	layout := scope.layout(layers)
	rep.TargetBase = displayPath(layout.root)
	builder, err := source.StandardDotfilesOptions{
		ConfigTreeDir:    layout.configTreeDir,
		ConfigTreeTarget: layout.root,
		ModulesDir:       layout.modulesDir,
		ModulesCfg:       cfg,
		Extra:            layout.extra,
	}.Builder(cfg)
	if err != nil {
		rep.Err = fmt.Errorf("build %s sources: %w", scope, err)

		return rep, rep.Err
	}
	// The pipeline fans modules out through a taskgroup, which only the
	// caller can open: a request context carries none.
	var built *source.Tree
	err = taskgroup.WithSession(ctx, func(sessionCtx context.Context) error {
		built, err = builder.Tree(sessionCtx)

		return err
	})
	if err != nil {
		rep.Err = fmt.Errorf("compose %s files: %w", scope, err)

		return rep, rep.Err
	}
	rep.Warnings = built.Warnings
	rep.Files = materialFiles(built.Files())

	return rep, nil
}

// scopeLayout is where a scope's apply reads and writes, mirroring what
// `modot <scope> apply` passes to the source builder.
type scopeLayout struct {
	// root is the apply root: the home directory, the workspace, or /.
	root string
	// modulesDir is the directory holding the module checkouts.
	modulesDir string
	// configTreeDir is the direct config tree; "" when the scope has none.
	configTreeDir string
	// extra plugins run before the config tree and module scanners.
	extra []source.Plugin
}

// layout resolves the paths this scope's apply would use. It differs per scope
// because the commands do: home applies under ~, codebase under the workspace
// that holds modot.cue, and system under /.
func (s Scope) layout(layers []configcue.Layer) scopeLayout {
	modulesDir := modulesRoot(layers)
	switch s {
	case ScopeHome:
		return scopeLayout{
			root:          homeDir(),
			modulesDir:    modulesDir,
			configTreeDir: filepath.Join(filepath.Dir(modulesDir), "config"),
			// home apply places dconf values, so the tree includes the
			// plugin that turns them into files.
			extra: []source.Plugin{&apply.DconfPlugin{}},
		}
	case ScopeSystem:
		return scopeLayout{
			root:          string(filepath.Separator),
			modulesDir:    modulesDir,
			configTreeDir: "",
			extra:         nil,
		}
	case ScopeCodebase:
		// codebase apply reads .modot/config under the workspace root and
		// modules/ beside it.
		root := filepath.Dir(modulesDir)
		if len(layers) > 0 {
			root = filepath.Dir(layers[0].Path)
		}

		configTreeDir := filepath.Join(root, ".modot", "config")
		info, statErr := os.Stat(configTreeDir)
		if statErr != nil || !info.IsDir() {
			configTreeDir = ""
		}

		return scopeLayout{
			root:          root,
			modulesDir:    modulesDir,
			configTreeDir: configTreeDir,
			extra:         nil,
		}
	}

	return scopeLayout{root: homeDir(), modulesDir: modulesDir, configTreeDir: "", extra: nil}
}

// homeDir is the user's home directory, or "" when it cannot be resolved. An
// empty root still composes; only the display of absolute paths suffers.
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return home
}

// materialFiles turns the composed tree into rows. The body is read here
// because the tree is rebuilt on every request: a second request cannot reach
// this one's File handles, so a body not read now is not available at all.
func materialFiles(files []source.File) []MaterialFile {
	out := make([]MaterialFile, 0, len(files))
	for _, file := range files {
		rel := filepath.ToSlash(file.RelPath())
		target := filepath.Join(file.TargetBase(), file.RelPath())
		content, contentErr := readContent(file)
		if contentErr != nil {
			slog.Warn("dashboard: read file", "path", rel, "error", contentErr)
			content = FileContent{Text: "", Size: 0, Lines: 0, Truncated: false, Err: contentErr}
		}
		out = append(out, MaterialFile{
			Target:  displayPath(target),
			Path:    rel,
			Dir:     directoryOf(rel),
			Name:    filepath.Base(rel),
			Root:    displayPath(file.TargetBase()),
			Kind:    file.Type().String(),
			Source:  sourceLabel(file),
			Mode:    fmt.Sprintf("%04o", file.Mode().Perm()),
			Content: content,
		})
	}

	return out
}

// sourceLabel says what produced a file. A module wins over the rest, so a
// module's template is credited to the module and not to the pipeline stage.
// A place entry names its key, which is the path a reader can find in the
// config and jump to in the merged tree.
func sourceLabel(file source.File) string {
	if scoped, ok := file.(source.ScopedFile); ok && scoped.ModuleName() != "" {
		return scoped.ModuleName()
	}
	info := file.SourceInfo()
	if key, found := strings.CutPrefix(info, "place:"); found {
		return key
	}

	return info
}

// directoryOf is a relative path's directory, "" for a file at the root.
func directoryOf(path string) string {
	index := strings.LastIndex(path, "/")
	if index < 0 {
		return ""
	}

	return path[:index]
}

// displayPath abbreviates the home directory to ~, so targets read the way the
// config spells them.
func displayPath(path string) string {
	if path == "" {
		return ""
	}
	home := homeDir()
	if home == "" || home == string(filepath.Separator) {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(filepath.Separator)
	if rest, found := strings.CutPrefix(path, prefix); found {
		return "~/" + filepath.ToSlash(rest)
	}

	return path
}

// MaterialCount is how many files the scope would place.
func (rep FilesReport) MaterialCount() int { return len(rep.Files) }

// KindCounts counts the files per source kind, ordered by count then name.
func (rep FilesReport) KindCounts() []FileCount {
	counts := map[string]int{}
	for _, file := range rep.Files {
		counts[file.Kind]++
	}

	return sortedCounts(counts)
}

// SourceCounts counts the files per producing source, most first.
func (rep FilesReport) SourceCounts() []FileCount {
	counts := map[string]int{}
	for _, file := range rep.Files {
		name := file.Source
		if name == "" {
			name = "unknown"
		}
		counts[name]++
	}

	return sortedCounts(counts)
}

// FileCount is one label with a count, for the summaries.
type FileCount struct {
	Label string
	Count int
}

// sortedCounts orders a count map by count descending, then label.
func sortedCounts(counts map[string]int) []FileCount {
	out := make([]FileCount, 0, len(counts))
	for label, count := range counts {
		out = append(out, FileCount{Label: label, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}

		return out[i].Label < out[j].Label
	})

	return out
}
