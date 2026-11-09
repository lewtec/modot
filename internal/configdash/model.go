package configdash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

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

// Scope selects which layers get discovered.
type Scope string

const (
	// ScopeHome discovers dotfiles, user, and home layers (modot home config).
	ScopeHome Scope = "home"
	// ScopeCodebase discovers the repo layer (modot codebase config).
	ScopeCodebase Scope = "codebase"
	// ScopeSystem discovers dotfiles, user, home, and host layers for the
	// system tree (modot system config).
	ScopeSystem Scope = "system"
)

// DiscoverOptions maps the scope onto the loader's discovery options.
func (s Scope) DiscoverOptions(cwd string) configcue.DiscoverOptions {
	switch s {
	case ScopeHome:
		return configcue.DiscoverOptions{Cwd: cwd, Mode: filespine.ModeHome, HomeLayers: true}
	case ScopeSystem:
		return configcue.DiscoverOptions{Cwd: cwd, Mode: filespine.ModeSystem, HomeLayers: true}
	case ScopeCodebase:
		return configcue.DiscoverOptions{Cwd: cwd, Mode: filespine.ModeCodebase, HomeLayers: false}
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
	// Files maps a leaf path to the file CUE recorded for the assertion
	// that won it, for every leaf the evaluation could attribute. Sources
	// fills the gaps (a leaf with no record still belongs to its module).
	Files map[string]string
	// Builtin maps a leaf path to the embedded layer that set it (schema,
	// a prelude, the runtime). A builtin leaf still has an origin; it is a
	// default, not a value nobody can account for.
	Builtin map[string]string
	// Sources maps a dotted key path to the module that owns it. A key with
	// no entry comes from the entry layers, the schema, or the runtime.
	Sources map[string]Source
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
	// Source is the provider the module comes from: "core" for a built-in,
	// "self" for a directory in this workspace, or a fetched provider.
	Source string
	// Ref is the module name inside Source: the core module name, or the
	// directory a self module is read from.
	Ref string
	// Err is set when this module's declaration could not be read.
	Err error
	// Problem is set when the module cannot be read here: a local module
	// missing on disk, a fragment with a broken options block, or a built-in
	// module that has no fragment at all (which is not an error).
	Problem *ModuleProblem
	// Unavailable is set when Err is this module's fault (it did not load).
	// It is false for a module that simply has no fragment on disk.
	Unavailable bool
}

// ModuleProblem is why a module could not be read, in the order a reader
// needs it: what was tried, why it failed, and what to change.
type ModuleProblem struct {
	// Provider names the module's source ("core", a fetch) when it has no
	// file on disk at all; empty for a local module missing from disk.
	Provider string
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

// Where names a file the way a reader recognizes it. Every config file in modot
// is called modot.cue, so a basename says nothing: the label is the layer name
// for an entry file, <module> for a fragment under modules/, and the path
// relative to the workspace otherwise.
func (rep Report) Where(file string) string {
	if file == "" {
		return ""
	}
	for _, layer := range rep.Layers {
		if filepath.Clean(layer.Path) == filepath.Clean(file) {
			return layer.Name
		}
	}
	if rep.Root != "" {
		modulesDir := filepath.Dir(rep.Root)
		rel, err := filepath.Rel(modulesDir, file)
		if err == nil && !strings.HasPrefix(rel, "..") {
			// modules/<name>/modot.cue reads as <name>.
			if dir, _, ok := strings.Cut(filepath.ToSlash(rel), "/"); ok {
				return dir
			}
		}
	}
	rel, err := filepath.Rel(filepath.Dir(rep.Root), file)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}

	return filepath.Base(file)
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
		Files:     nil,
		Builtin:   nil,
	}
	result, err := configcue.Evaluate(ctx, opts)
	if err != nil {
		err = fmt.Errorf("evaluate %s layers: %w", scope, err)
		// The layers are still worth showing next to the merge error.
		rep.MergedErr = err
		rep.Modules = inspectModules(nil, rep.Root, nil, nil)

		return rep, err
	}
	err = rep.fill(result)
	if err != nil {
		return Report{}, err
	}

	return rep, moduleErrors(rep.Modules)
}

// ProblemCount counts the modules the page explains instead of showing a
// schema: the ones that failed, and the built-ins with no fragment here.
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
	cfg, err := configcue.ConfigFromJSON(result.JSON)
	if err != nil {
		slog.Debug("decode config for module inspection", "error", err)
	}
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
	rep.Modules = inspectModules(cfg, rep.Root, moduleEntries(rep.Merged), options)
	rep.Sources = sources(rep.Merged, rep.Modules)
	provenance := result.Provenance()
	rep.Files = provenance.Files
	rep.Builtin = provenance.Builtin
	for path, file := range rep.Files {
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

	return nil
}

// moduleErrors joins the per-module read failures.
func moduleErrors(modules []Module) error {
	var err error
	for _, mod := range modules {
		if mod.Unavailable && mod.Err != nil {
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
func inspectModules(
	cfg *configcue.Config,
	root string,
	entries map[string]configcue.ModuleEntry,
	options map[string]any,
) []Module {
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
		out = append(out, inspectModule(cfg, root, name, entries[name], options))
	}

	return out
}

// inspectModule reads one module. It never fails: a module that cannot be read
// carries Err so the UI can show it. The path comes from the loader's own
// resolver, so the dashboard reads exactly the file the merge reads.
func inspectModule(
	cfg *configcue.Config,
	root, name string,
	entry configcue.ModuleEntry,
	options map[string]any,
) Module {
	mod := Module{Name: name, Enable: entry.Enable, Path: root, File: "", RootKeys: nil,
		OptionsCUE: "", Merged: nil, Source: "", Ref: "", Err: nil, Problem: nil,
		Unavailable: false}
	if options != nil {
		mod.Merged, _ = options[name].(map[string]any)
	}
	if cfg != nil {
		mod.Source, mod.Ref = moduleSource(name, entry)
	}
	modulePath, local, err := modulePath(cfg, name, entry, root)
	switch {
	case err != nil:
		mod.Err = fmt.Errorf("%w: %w", ErrModuleOptions, err)
		mod.Unavailable = true
		mod.Problem = &ModuleProblem{
			Summary:  "module source not resolvable",
			Provider: "",
			Looked:   "",
			Declared: declaredEntry(name, entry),
			Hint:     "",
			Detail:   err.Error(),
		}

		return mod
	case !local:
		// core: and fetched modules have no file here; their options are
		// whatever the config wrote under options.<name>.
		mod.Path = ""
		mod.File = ""
		mod.Problem = nonLocalProblem(mod)

		return mod
	}

	return readModule(mod, modulePath, name, entry)
}

// readModule reads a local module's fragment and its declared options.
func readModule(mod Module, modulePath, name string, entry configcue.ModuleEntry) Module {
	mod.Path = modulePath
	mod.File = modulecue.FilePath(mod.Path)
	if !modulecue.Exists(mod.Path) {
		mod.Err = fmt.Errorf("%w: %s", ErrModuleFile, mod.File)
		mod.Unavailable = true
		mod.Problem = missingModuleProblem(name, mod.File, entry)

		return mod
	}
	expr, err := modulecue.OptionsSyntaxWithRoot(mod.Path, nil)
	if err != nil {
		mod.Err = fmt.Errorf("%w: %w", ErrModuleOptions, err)
		mod.Unavailable = true
		mod.Problem = &ModuleProblem{
			Summary:  "module options not readable",
			Provider: "",
			Looked:   mod.File,
			Declared: "",
			Hint: "Fix the options block in " + mod.File + "; `modot home|codebase config eval` " +
				"shows the same error without the dashboard.",
			Detail: err.Error(),
		}

		return mod
	}
	mod.OptionsCUE = expr
	mod.RootKeys = rootKeys(expr)

	return mod
}

// modulePath resolves a module through the loader. local is false when the
// module comes from a provider with no directory on disk (core:, a fetch).
func modulePath(cfg *configcue.Config, name string, entry configcue.ModuleEntry, root string) (string, bool, error) {
	if cfg == nil || root == "" {
		return moduleDir(root, name, entry), true, nil
	}
	path, ok, err := configcue.ResolveModulePath(cfg, name, entry, root)
	if err != nil {
		return "", false, fmt.Errorf("resolve modules.%s: %w", name, err)
	}
	if !ok {
		return "", false, nil
	}

	return path, true, nil
}

// moduleSource names where a module comes from and what it points at there:
// "core" and the core module name, "self" and the directory, or a fetch.
func moduleSource(name string, entry configcue.ModuleEntry) (string, string) {
	from := strings.TrimSpace(entry.From)
	if from == "" {
		from = strings.TrimSpace(entry.Input)
	}
	if from == "" {
		from = "self"
	}
	provider, ref, ok := strings.Cut(from, ":")
	if !ok {
		return strings.TrimSpace(from), name
	}
	provider = strings.TrimSpace(provider)
	ref = strings.TrimSpace(ref)
	if provider == "self" {
		// A self: input is the module directory; an explicit path is where
		// the module lives inside it.
		if path := strings.Trim(strings.TrimSpace(entry.Path), "/"); path != "" {
			ref = strings.Trim(path, "/")
		}

		return provider, ref
	}

	return provider, ref
}

// nonLocalProblem says a module has no fragment on disk here, which is normal
// for core: and fetched modules.
func nonLocalProblem(mod Module) *ModuleProblem {
	return &ModuleProblem{
		Summary:  "built-in module, no fragment on disk",
		Provider: mod.Source,
		Looked:   "",
		Declared: "",
		Hint: fmt.Sprintf("%s modules are built into modot: %s:%s has no modot.cue to read. "+
			"Its options are whatever the config sets under options.%s.",
			mod.Source, mod.Source, mod.Ref, mod.Name),
		Detail: "",
	}
}

// declaredEntry renders the config line that named a module.
func declaredEntry(name string, entry configcue.ModuleEntry) string {
	switch {
	case entry.Input != "":
		return fmt.Sprintf("modules.%s.input = %q", name, entry.Input)
	case entry.Path != "":
		return fmt.Sprintf("modules.%s.path = %q", name, entry.Path)
	default:
		return ""
	}
}

// moduleDir is the directory a module is read from when the loader cannot
// resolve it (no config was decoded): modules/<path>, or modules/<name>.
func moduleDir(root, name string, entry configcue.ModuleEntry) string {
	workspace := filepath.Dir(root)
	modulePath := filepath.ToSlash(filepath.Join("modules", name))
	if explicit := strings.Trim(strings.TrimSpace(entry.Path), "/"); explicit != "" {
		modulePath = filepath.ToSlash(filepath.Join("modules", explicit))
	}

	return filepath.Join(workspace, filepath.FromSlash(modulePath))
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
		Provider: "",
		Summary:  "module file not found",
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
	if strings.HasPrefix(strings.TrimSpace(entry.Input), "self:") {
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

// walkLeaves calls visit for every scalar path in node, in CUE's path form:
// lists use the index in brackets, as in backup.actions[0].name.
func walkLeaves(node map[string]any, visit func(path string)) {
	for key, value := range node {
		walkValue(key, value, visit)
	}
}

// walkValue visits the scalars under value, recursing through structs and
// lists. CUE writes a list element as parent[index], so provenance and owner
// lookups use that form rather than a dotted index.
func walkValue(prefix string, value any, visit func(path string)) {
	switch container := value.(type) {
	case map[string]any:
		for key, child := range container {
			walkValue(joinField(prefix, key), child, visit)
		}
	case []any:
		for i, child := range container {
			walkValue(indexPath(prefix, i), child, visit)
		}
	default:
		visit(prefix)
	}
}

// indexPath is CUE's path for a list element: parent[0].
func indexPath(prefix string, index int) string {
	return prefix + "[" + strconv.Itoa(index) + "]"
}

// joinField appends a field the way CUE spells it: a bare identifier, or a
// quoted label when the key is not one. Paths from CUE and paths built here are
// compared as strings, so file.home.".agents/skills".type must get the quotes.
func joinField(prefix, key string) string {
	if isCUEIdent(key) {
		return join(prefix, key)
	}

	return join(prefix, strconv.Quote(key))
}

// isCUEIdent reports whether key is a plain CUE identifier, which needs no
// quoting in a path.
func isCUEIdent(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) {
			continue
		}

		return false
	}

	return true
}

func count(merged map[string]any, src map[string]Source) Stats {
	stats := Stats{Keys: 0, FromModules: 0, FromRoot: 0, Conflicted: 0}
	var walk func(prefix string, node any)
	walk = func(prefix string, node any) {
		switch container := node.(type) {
		case map[string]any:
			if len(container) == 0 {
				stats.Keys++
			}
			for key, value := range container {
				walk(join(prefix, key), value)
			}

			return
		case []any:
			if len(container) == 0 {
				stats.Keys++
			}
			for i, value := range container {
				walk(indexPath(prefix, i), value)
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
