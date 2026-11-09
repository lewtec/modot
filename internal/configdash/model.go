package configdash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

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
		return configcue.DiscoverOptions{Cwd: cwd, HomeLayers: true, Mode: filespine.ModeHome}
	}
	return configcue.DiscoverOptions{Cwd: cwd, Mode: filespine.ModeCodebase}
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
	Merged map[string]any
	// Err is set when this module's declaration could not be read.
	Err error
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
		return nil, err
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
		Scope:  scope,
		Mode:   opts.RuntimeMode(),
		Root:   modulesRoot(layers),
		Layers: layers,
	}
	result, err := configcue.Evaluate(ctx, opts)
	if err != nil {
		// The layers are still worth showing next to the merge error.
		rep.MergedErr = err
		rep.Modules = inspectModules(rep.Root, nil, nil)
		return rep, err
	}
	if err := rep.fill(result); err != nil {
		return Report{}, err
	}
	return rep, moduleErrors(rep.Modules)
}

// fill decodes the evaluated Configuration into rep and records where each
// merged key came from.
func (rep *Report) fill(result configcue.EvaluationResult) error {
	if len(result.JSON) > 0 {
		if err := json.Unmarshal(result.JSON, &rep.Merged); err != nil {
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
		entry := configcue.ModuleEntry{Enable: true}
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
	mod := Module{Name: name, Enable: entry.Enable, Path: root}
	if entry.Path != "" {
		mod.Path = filepath.Join(root, entry.Path)
	} else if root != "" {
		mod.Path = filepath.Join(root, name)
	}
	mod.File = modulecue.FilePath(mod.Path)
	if options != nil {
		mod.Merged, _ = options[name].(map[string]any)
	}
	if root == "" || !modulecue.Exists(mod.Path) {
		mod.Err = fmt.Errorf("%w: %s", ErrModuleFile, mod.File)
		return mod
	}
	expr, err := modulecue.OptionsSyntaxWithRoot(mod.Path, nil)
	if err != nil {
		mod.Err = err
		return mod
	}
	mod.OptionsCUE = expr
	mod.RootKeys = rootKeys(expr)
	return mod
}

// sources maps dotted key paths to the module that owns them. A module owns
// options.<name>, which is the shared tree the loader builds from its schema,
// and any top-level key its options schema declares (a module may declare a
// root field for another module to read).
func sources(merged map[string]any, modules []Module) map[string]Source {
	out := map[string]Source{}
	owners := rootOwners(modules)
	walkLeaves(merged, func(path string) {
		out[path] = Source{Module: ownerOf(path, owners)}
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

// walkLeaves calls fn for every scalar path in node.
func walkLeaves(node map[string]any, fn func(path string)) {
	var walk func(prefix string, value any)
	walk = func(prefix string, value any) {
		if obj, ok := value.(map[string]any); ok {
			for key, child := range obj {
				walk(join(prefix, key), child)
			}
			return
		}
		fn(prefix)
	}
	for key, value := range node {
		walk(key, value)
	}
}

func count(merged map[string]any, src map[string]Source) Stats {
	stats := Stats{}
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
