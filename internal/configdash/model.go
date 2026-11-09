package configdash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/filespine"
	"github.com/lewtec/modot/internal/modulecue"
)

var (
	// ErrNoConfig means discovery found no modot.cue for the scope.
	ErrNoConfig = errors.New("no modot.cue found")
	// ErrModuleFile means a named module has no modot.cue on disk.
	ErrModuleFile = errors.New("module file not found")
	// ErrModuleOptions means the module's modot.cue has no readable options.
	ErrModuleOptions = errors.New("module options not readable")
)

// missingFileSummary is the ModuleProblem summary for ErrModuleFile.
const missingFileSummary = "module file not found"

// Scope selects which layers get discovered.
type Scope string

const (
	// ScopeHome discovers dotfiles, user, and home layers (modot home config).
	ScopeHome Scope = "home"
	// ScopeCodebase discovers the repo layer (modot codebase config).
	ScopeCodebase Scope = "codebase"
)

// DiscoverOptions maps the scope onto the loader's discovery options.
func (s Scope) DiscoverOptions(cwd string) configcue.DiscoverOptions {
	if s == ScopeHome {
		return configcue.DiscoverOptions{Cwd: cwd, Mode: filespine.ModeHome, HomeLayers: true}
	}

	return configcue.DiscoverOptions{Cwd: cwd, Mode: filespine.ModeCodebase, HomeLayers: false}
}

// Report is one evaluated Configuration plus the modules that built it.
type Report struct {
	Scope  Scope
	Mode   string
	Root   string
	Layers []configcue.Layer
	// Modules is the TEC-01 set, enabled and disabled, sorted by name.
	Modules []Module
	Merged  map[string]any
	// MergedErr is set when the final merge failed. Layers and Modules are
	// still filled so the dashboard can show them next to the error.
	MergedErr error
	Stats     Stats
	// Sources maps a dotted key path to the module that owns it. A key with
	// no entry comes from the entry layers, the schema, or the runtime.
	Sources map[string]Source
	// Unused are module directories on disk that no modules.<name> entry
	// points at. The entry file decides what loads; these are just candidates.
	Unused []string
}

// Module is one module named by the entry file.
type Module struct {
	Name string
	// Path is the module directory; File is its modot.cue.
	Path   string
	File   string
	Enable bool
	// RootKeys are the top-level fields the module's options schema declares.
	RootKeys []string
	// OptionsCUE is the declared options expression, formatted as CUE.
	OptionsCUE string
	// Merged is what options.<name> resolved to in the final Configuration.
	// It is nil when the module's options never made it into the merge.
	Merged map[string]any
	// Err is set when this module's declaration could not be read.
	Err error
	// Problem explains an Err in terms a user can act on.
	Problem *ModuleProblem
}

// ModuleProblem is why a module could not be read, in the order a reader
// needs it: what was tried, why it failed, and what to change.
type ModuleProblem struct {
	// Summary is the one-line cause, e.g. "module file not found".
	Summary string
	// Looked is the path that was not there (missing files only).
	Looked string
	// Declared is the modules.<name> entry as the config wrote it, empty when
	// the entry names no path.
	Declared string
	// Hint is the fix: which config key to edit and what to set it to.
	Hint string
	// Detail is the underlying error text (parse errors only).
	Detail string
}

// Source names where a merged key came from. Module is set when a module
// supplied it; File is the exact file CUE recorded for the value.
type Source struct {
	Module string
	File   string
}

// Owned reports whether a module supplied the key.
func (s Source) Owned() bool { return s.Module != "" }

// Filename returns the base name of the file that set the key.
func (s Source) Filename() string {
	if s.File == "" {
		return ""
	}

	return filepath.Base(s.File)
}

// Stats are the dashboard counters.
type Stats struct {
	Keys        int
	FromModules int
	FromRoot    int
	Conflicted  int
}

// Discover returns the layers scope would load from cwd, without evaluating
// them. It is the cheap way to tell which scope a directory belongs to.
func Discover(ctx context.Context, scope Scope, cwd string) ([]configcue.Layer, error) {
	discovered, err := configcue.DiscoverLayers(ctx, scope.DiscoverOptions(cwd))
	if err != nil {
		return nil, fmt.Errorf("discover %s layers: %w", scope, err)
	}
	if len(discovered.Layers) == 0 {
		return nil, ErrNoConfig
	}

	return discovered.Layers, nil
}

// Load evaluates the Configuration for scope starting at cwd. A non-nil error
// may still come with a Report: the layers and modules are known even when the
// merge failed, so a caller can show them next to the error.
func Load(ctx context.Context, scope Scope, cwd string) (Report, error) {
	opts := scope.DiscoverOptions(cwd)
	layers, err := Discover(ctx, scope, cwd)
	if err != nil {
		return Report{}, err
	}

	rep := Report{
		Scope:     scope,
		Mode:      opts.RuntimeMode(),
		Root:      modulesRoot(layers),
		Layers:    layers,
		Modules:   nil,
		Merged:    nil,
		MergedErr: nil,
		Stats:     Stats{Keys: 0, FromModules: 0, FromRoot: 0, Conflicted: 0},
		Sources:   nil,
		Unused:    nil,
	}
	result, err := configcue.Evaluate(ctx, opts)
	if err != nil {
		err = fmt.Errorf("evaluate %s layers: %w", scope, err)
		// The layers are still worth showing next to the merge error.
		rep.MergedErr = err
		rep.Modules = inspectModules(rep.Root, nil, nil)

		return rep, err
	}
	err = rep.fill(result)
	if err != nil {
		return Report{}, err
	}

	return rep, moduleErrors(rep.Modules)
}

// ProblemCount counts the modules that could not be read.
func (rep Report) ProblemCount() int {
	problems := 0
	for _, mod := range rep.Modules {
		if mod.Problem != nil {
			problems++
		}
	}

	return problems
}

// fill decodes the evaluated Configuration into rep and records where each
// merged key came from.
func (rep *Report) fill(result configcue.EvaluationResult) error {
	if len(result.JSON) > 0 {
		err := json.Unmarshal(result.JSON, &rep.Merged)
		if err != nil {
			return fmt.Errorf("decode evaluated config: %w", err)
		}
	}
	if rep.Merged == nil {
		rep.Merged = map[string]any{}
	}
	options, _ := rep.Merged["options"].(map[string]any)
	rep.Modules = inspectModules(rep.Root, moduleEntries(rep.Merged), options)
	rep.Sources = sources(rep.Merged, rep.Modules)
	for path, file := range result.Provenance().Files {
		source := rep.Sources[path]
		source.File = file
		// options.<name> belongs to the module that declared the schema even
		// when another layer supplied the concrete value: the module is what
		// makes the key exist.
		if source.Module == "" {
			source.Module = moduleForFile(rep.Modules, file)
		}
		rep.Sources[path] = source
	}
	rep.Stats = count(rep.Merged, rep.Sources)
	rep.Unused = unusedModules(rep.Root, rep.Modules)

	return nil
}

// moduleErrors joins the per-module read failures.
func moduleErrors(modules []Module) error {
	var err error
	for _, mod := range modules {
		if mod.Err != nil {
			err = errors.Join(err, mod.Err)
		}
	}

	return err
}

// moduleForFile returns the module name whose modot.cue is file, or "".
func moduleForFile(modules []Module, file string) string {
	for _, mod := range modules {
		if mod.File == file {
			return mod.Name
		}
	}

	return ""
}

// modulesRoot is the modules/ directory of the layer that owns the entry file.
func modulesRoot(discovered []configcue.Layer) string {
	for _, layer := range discovered {
		if layer.Name == "repo" || layer.Name == "dotfiles" {
			return filepath.Join(filepath.Dir(layer.Path), "modules")
		}
	}
	if len(discovered) == 0 {
		return ""
	}

	return filepath.Join(filepath.Dir(discovered[0].Path), "modules")
}

// moduleEntries decodes modules.<name> from the merged Configuration.
func moduleEntries(merged map[string]any) map[string]configcue.ModuleEntry {
	entries := map[string]configcue.ModuleEntry{}
	raw, _ := merged["modules"].(map[string]any)
	for name, value := range raw {
		obj, ok := value.(map[string]any)
		if !ok {
			continue
		}
		entry := configcue.ModuleEntry{Enable: true, Input: "", Path: "", From: "", Version: ""}
		if v, ok := obj["enable"].(bool); ok {
			entry.Enable = v
		}
		entry.Input, _ = obj["input"].(string)
		entry.Path, _ = obj["path"].(string)
		entries[name] = entry
	}

	return entries
}

// inspectModules reads each module's declared options. It never fails: a
// module that cannot be read carries Err so the UI can show it.
func inspectModules(root string, entries map[string]configcue.ModuleEntry, options map[string]any) []Module {
	if len(entries) == 0 {
		return nil
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Module, 0, len(names))
	for _, name := range names {
		out = append(out, inspectModule(root, name, entries[name], options))
	}

	return out
}

// inspectModule reads one module. It never fails: a module that cannot be read
// carries Err so the UI can show it.
func inspectModule(root, name string, entry configcue.ModuleEntry, options map[string]any) Module {
	mod := Module{Name: name, Enable: entry.Enable, Path: root, File: "", RootKeys: nil,
		OptionsCUE: "", Merged: nil, Err: nil, Problem: nil}
	mod.Path = moduleDir(root, name, entry)
	mod.File = modulecue.FilePath(mod.Path)
	if options != nil {
		mod.Merged, _ = options[name].(map[string]any)
	}
	if root == "" || !modulecue.Exists(mod.Path) {
		mod.Err = fmt.Errorf("%w: %s", ErrModuleFile, mod.File)
		mod.Problem = missingModuleProblem(name, mod.File, entry)

		return mod
	}
	expr, err := modulecue.OptionsSyntaxWithRoot(mod.Path, nil)
	if err != nil {
		mod.Err = fmt.Errorf("%w: %w", ErrModuleOptions, err)
		mod.Problem = &ModuleProblem{
			Summary:  "module options not readable",
			Looked:   mod.File,
			Declared: "",
			Detail:   err.Error(),
			Hint: "Fix the options block in " + mod.File + "; `modot home|codebase config eval` " +
				"shows the same error without the dashboard.",
		}

		return mod
	}
	mod.OptionsCUE = expr
	mod.RootKeys = rootKeys(expr)

	return mod
}

// unusedModules lists the module directories that exist in the module tree but
// that no declared module resolves to. Nothing points at them, so they do not
// load; the entry file decides what does, and the dashboard only says so.
func unusedModules(root string, modules []Module) []string {
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	used := usedDirs(modules)
	unused := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if !used[dir] {
			unused = append(unused, dir)
		}
	}
	sort.Strings(unused)

	return unused
}

// usedDirs maps the directory each module is read from. A module that is
// missing where the loader looked still claims its path: the problem is
// reported, not the directory.
func usedDirs(modules []Module) map[string]bool {
	used := make(map[string]bool, len(modules))
	for _, mod := range modules {
		// A module from another provider (core:, github:) has no local
		// directory; its would-be path must not mark one as used. Anything
		// else, even a module that failed to read, owns its directory.
		missing := mod.Problem != nil && mod.Problem.Summary == missingFileSummary
		if !missing {
			used[mod.Path] = true
		}
	}

	return used
}

// moduleDir is the directory a module is read from: the workspace joined with
// its `self:` input (or `from` ref) when the entry names one, modules/<path>
// when it names a path, modules/<name> otherwise. A dashed name cannot be a
// modules/<name> directory without a path or input saying so, which is the
// usual cause of "module file not found".
func moduleDir(root, name string, entry configcue.ModuleEntry) string {
	workspace := filepath.Dir(root)
	// Without any hint the directory is modules/<name>, which only exists for
	// CUE-safe names.
	modulePath := filepath.ToSlash(filepath.Join("modules", name))
	explicit := strings.Trim(strings.TrimSpace(entry.Path), "/")
	if explicit != "" {
		modulePath = filepath.ToSlash(filepath.Join("modules", explicit))
	}
	if ref, ok := selfRef(entry.Input); ok && ref != "" && !explicitLocal(explicit) {
		// A `self:` input is the module directory relative to the workspace;
		// a path only names the module inside it.
		modulePath = filepath.ToSlash(filepath.Join(ref, explicit))
	}

	return filepath.Join(workspace, filepath.FromSlash(modulePath))
}

// selfRef returns the workspace-relative directory a spec names: `self:x`
// names x and bare `self` names the workspace root. Anything else (a fetch, a
// named input) is not local and reports ok=false.
func selfRef(spec string) (string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "self" || spec == "" {
		return "", true
	}
	if !strings.HasPrefix(spec, "self:") {
		return "", false
	}

	return strings.Trim(strings.TrimPrefix(spec, "self:"), "/"), true
}

// explicitLocal reports whether a path already starts at the workspace, as in
// modules/<dir>, which a self: input must not be joined to again.
func explicitLocal(path string) bool {
	return strings.HasPrefix(path, "modules/") || strings.HasPrefix(path, "./")
}

// specLocal reports whether an input names a workspace directory (`self:`),
// as opposed to the bare `self` that means the workspace root itself.
func specLocal(spec string) bool {
	return strings.HasPrefix(strings.TrimSpace(spec), "self:")
}

// missingModuleProblem explains a module whose file is not on disk: the
// directory that was tried, the modules.<name> declaration it came from, and
// the two ways to fix it.
func missingModuleProblem(name, file string, entry configcue.ModuleEntry) *ModuleProblem {
	declared := ""
	if entry.Path != "" {
		declared = fmt.Sprintf("modules.%s.path = %q", name, entry.Path)
	} else if entry.Input != "" {
		declared = fmt.Sprintf("modules.%s.input = %q", name, entry.Input)
	}

	return &ModuleProblem{
		Summary:  missingFileSummary,
		Looked:   file,
		Declared: declared,
		Detail:   "",
		Hint:     missingModuleHint(name, file, entry),
	}
}

// missingModuleHint names the fix. A dashed directory next to the one tried is
// the usual cause: CUE selects options.<name>, so a module declared
// base16_dunst looks for modules/base16_dunst while the checkout holds
// modules/base16-dunst.
func missingModuleHint(name, file string, entry configcue.ModuleEntry) string {
	dashed, found := dashedSibling(name, file)
	if !found {
		return fmt.Sprintf("Point modules.%s.path at the module directory, the one holding "+
			"modot.cue (tried %s), or set modules.%s.enable = false to stop loading it.",
			name, filepath.Dir(file), name)
	}
	dir := filepath.Base(dashed)
	// The fix is the line that already names a directory: the input that
	// points at the wrong one, or a path when nothing names one.
	fix := fmt.Sprintf(`modules.%s.path = "modules/%s"`, name, dir)
	if specLocal(entry.Input) {
		fix = fmt.Sprintf(`modules.%s.input = "self:modules/%s"`, name, dir)
	}

	return fmt.Sprintf("The loader looked in %s. %s exists; set %s. "+
		"The name keeps the underscore: CUE cannot select options.%s with a dash "+
		"in the field name.", filepath.Dir(file), dashed, fix, strings.ReplaceAll(name, "_", "-"))
}

// dashedSibling finds a module directory whose name swaps the underscores of
// name for dashes, next to the directory tried for file: modules/base16-dunst
// for modules/base16_dunst.
func dashedSibling(name, file string) (string, bool) {
	dashed := strings.ReplaceAll(name, "_", "-")
	if dashed == name {
		return "", false
	}
	other := filepath.Join(filepath.Dir(file), "..", dashed)
	info, err := os.Stat(other)

	return other, err == nil && info.IsDir()
}

// sources maps dotted key paths to the module that owns them. A module owns
// options.<name>, which is the shared tree the loader builds from its schema,
// and any top-level key its options schema declares (a module may declare a
// root field for another module to read).
func sources(merged map[string]any, modules []Module) map[string]Source {
	out := map[string]Source{}
	owners := rootOwners(modules)
	walkLeaves(merged, func(path string) {
		out[path] = Source{Module: ownerOf(path, owners), File: ""}
	})

	return out
}

// rootOwners maps a top-level key to the first enabled module whose options
// schema declares it. Only one module can own a key, per the loader.
func rootOwners(modules []Module) map[string]string {
	owners := map[string]string{}
	for _, mod := range modules {
		if !mod.Enable {
			continue
		}
		for _, key := range mod.RootKeys {
			if _, seen := owners[key]; !seen {
				owners[key] = mod.Name
			}
		}
	}

	return owners
}

// ownerOf names the module that owns a dotted path: options.<name> belongs to
// <name>, and any other path to the module declaring its root key.
func ownerOf(path string, owners map[string]string) string {
	root, rest, _ := strings.Cut(path, ".")
	if root == "options" && rest != "" {
		name, _, _ := strings.Cut(rest, ".")

		return name
	}

	return owners[root]
}

// walkLeaves calls visit for every scalar path in node.
func walkLeaves(node map[string]any, visit func(path string)) {
	var walk func(prefix string, value any)
	walk = func(prefix string, value any) {
		if obj, ok := value.(map[string]any); ok {
			for key, child := range obj {
				walk(join(prefix, key), child)
			}

			return
		}
		visit(prefix)
	}
	for key, value := range node {
		walk(key, value)
	}
}

func count(merged map[string]any, src map[string]Source) Stats {
	stats := Stats{Keys: 0, FromModules: 0, FromRoot: 0, Conflicted: 0}
	var walk func(prefix string, node any)
	walk = func(prefix string, node any) {
		if obj, ok := node.(map[string]any); ok {
			if len(obj) == 0 {
				stats.Keys++
			}
			for key, value := range obj {
				walk(join(prefix, key), value)
			}

			return
		}
		stats.Keys++
		if src[prefix].Owned() {
			stats.FromModules++
		} else {
			stats.FromRoot++
		}
	}
	for key, value := range merged {
		walk(key, value)
	}

	return stats
}

func join(prefix, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + "." + key
}
