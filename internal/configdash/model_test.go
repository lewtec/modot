package configdash_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/modot/internal/configdash"
	_ "github.com/lewtec/modot/internal/driver/prelude"
	"github.com/stretchr/testify/require"
)

// fixtureRoot is the workspace the load tests evaluate.
func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller")

	return filepath.Join(filepath.Dir(file), "testdata", "workspace")
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()
	t.Run("no config", func(t *testing.T) {
		t.Parallel()
		_, err := configdash.Load(t.Context(), configdash.ScopeCodebase, t.TempDir())
		require.ErrorIs(t, err, configdash.ErrNoConfig)
	})
	t.Run("merge failure still carries layers", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		// The schema types desktop.wallpaper as a struct.
		require.NoError(t, writeFile(root, "modot.cue", "package modot\n\ndesktop: wallpaper: \"not-a-struct\"\n"))
		rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
		require.Error(t, err)
		require.ErrorIs(t, err, rep.MergedErr)
		require.Len(t, rep.Layers, 1)
	})
}

func TestLoadDotfilesFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "Load")
	require.Len(t, rep.Layers, 1)
	require.Equal(t, "repo", rep.Layers[0].Name)
	require.Equal(t, filepath.Join(root, "modules"), rep.Root)

	byName := map[string]configdash.Module{}
	for _, mod := range rep.Modules {
		byName[mod.Name] = mod
	}
	require.Contains(t, byName, "theme")
	// Only the module's `options` schema is injected into the shared tree;
	// the rest of its fragment merges as ordinary config.
	require.Equal(t, []string{"accent"}, byName["theme"].RootKeys)
	require.True(t, byName["theme"].Enable)
	require.Equal(t, filepath.Join(root, "modules", "theme", "modot.cue"), byName["theme"].File)

	require.Equal(t, "theme", rep.Sources["options.theme.accent"].Module)
	// The entry file holds the literal, and it is the file that won.
	require.Equal(t, "modot.cue", rep.Sources["options.theme.accent"].Filename())
	// A root field merges as ordinary config, and the module file set it.
	require.Equal(t, "theme", rep.Sources["dashboard_demo.paper"].Module)
	require.Equal(t, "modot.cue", rep.Sources["dashboard_demo.paper"].Filename())
	// The runtime and generated layers are not user files and carry no file.
	require.Empty(t, rep.Sources["runtime.hostname"].File)

	// Leaf provenance points at the exact file that set each value, which is
	// what the merged tree renders per row.
	require.Equal(t, filepath.Join(root, "modot.cue"), rep.Files["options.theme.accent"])
	require.Equal(t, filepath.Join(root, "modules", "theme", "modot.cue"), rep.Files["dashboard_demo.paper"])
	require.NotContains(t, rep.Files, "runtime.hostname", "computed leaves have no file")
}

// Lists are config too. Provenance, owner, and counts must reach list elements,
// or a tree of backup actions stops at "[9 items]" with no file behind them.
func TestLoadListProvenance(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	config := "package modot\n\n" +
		"backup: actions: [\n" +
		"\t{name: \"cantgit\", kind: \"rsync\", src: \"/w/\", dst: \"host:w\"},\n" +
		"\t{name: \"camera\", kind: \"rsync\", src: \"/sdcard/\", dst: \"host:c\"},\n" +
		"]\n"
	require.NoError(t, writeFile(root, "modot.cue", config))

	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "Load")

	// CUE keys a list element as parent[index], not parent.index.
	require.Equal(t, filepath.Join(root, "modot.cue"), rep.Files["backup.actions[0].name"])
	require.Equal(t, filepath.Join(root, "modot.cue"), rep.Files["backup.actions[1].dst"])
	require.Contains(t, rep.Sources, "backup.actions[0].name")
	require.Contains(t, rep.Sources, "backup.actions[1].dst")
}

func TestLoadMissingModule(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, writeFile(root, "modot.cue", "package modot\n\nmodules: ghost: {path: \"missing\"}\n"))
	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.ErrorIs(t, err, configdash.ErrModuleFile)
	// The merge itself succeeded; only the module is unreadable.
	require.NoError(t, rep.MergedErr)
	require.Len(t, rep.Modules, 1)
	require.Equal(t, "ghost", rep.Modules[0].Name)
	require.ErrorIs(t, rep.Modules[0].Err, configdash.ErrModuleFile)
	require.Equal(t, 1, rep.ProblemCount())

	problem := rep.Modules[0].Problem
	require.NotNil(t, problem, "Problem")
	require.Equal(t, "module file not found", problem.Summary)
	// path is relative to the workspace, which is what the loader does.
	require.Equal(t, filepath.Join(root, "missing", "modot.cue"), problem.Looked)
	require.Equal(t, `modules.ghost.path = "missing"`, problem.Declared)
	require.Contains(t, problem.Hint, filepath.Join(root, "missing"))
	require.Contains(t, problem.Hint, "modules.ghost.enable = false")
}

// A module may be named with underscores while its checkout is dashed: CUE
// only selects options.<name>, so the name stays underscored and the path has
// to say where the directory is. The hint should name the directory that
// exists instead of only echoing the missing one.
func TestLoadDashedModuleDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	module := filepath.Join("modules", "base16-dunst")
	require.NoError(t, writeFile(root, "modot.cue",
		"package modot\n\nmodules: base16_dunst: {enable: true}\n"))
	require.NoError(t, os.MkdirAll(filepath.Join(root, module), 0o750))
	moduleCUE := "package base16_dunst\n\noptions: mode: *\"dark\" | string\n"
	require.NoError(t, writeFile(filepath.Join(root, module), "modot.cue", moduleCUE))

	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.ErrorIs(t, err, configdash.ErrModuleFile)
	require.Len(t, rep.Modules, 1)

	problem := rep.Modules[0].Problem
	require.NotNil(t, problem, "Problem")
	require.Equal(t, filepath.Join(root, "modules", "base16_dunst", "modot.cue"), problem.Looked)
	require.Contains(t, problem.Hint, filepath.Join(root, module))
	require.Contains(t, problem.Hint, `modules.base16_dunst.path = "modules/base16-dunst"`)
}

// A self: input is the module directory itself, so the loader finds it even
// when the name and the directory disagree on dashes.
func TestLoadDashedInputModule(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, writeFile(root, "modot.cue",
		"package modot\n\nmodules: base16_dunst: {input: \"self:modules/base16-dunst\", enable: true}\n"))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", "base16-dunst"), 0o750))
	require.NoError(t, writeFile(filepath.Join(root, "modules", "base16-dunst"), "modot.cue",
		"package base16_dunst\n\noptions: mode: *\"dark\" | string\n"))

	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err)
	require.Len(t, rep.Modules, 1)
	require.NoError(t, rep.Modules[0].Err)
	require.Equal(t, filepath.Join(root, "modules", "base16-dunst"), rep.Modules[0].Path)
	require.Equal(t, []string{"mode"}, rep.Modules[0].RootKeys)
	require.Equal(t, 0, rep.ProblemCount())
}

// A path with the dashes swapped is still a missing module, and the fix names
// the input that would find it.
func TestLoadDashedInputTypo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, writeFile(root, "modot.cue",
		"package modot\n\nmodules: base16_dunst: {input: \"self:modules/base16_dunst\", enable: true}\n"))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", "base16-dunst"), 0o750))
	require.NoError(t, writeFile(filepath.Join(root, "modules", "base16-dunst"), "modot.cue",
		"package base16_dunst\n\noptions: mode: *\"dark\" | string\n"))

	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.ErrorIs(t, err, configdash.ErrModuleFile)
	require.Equal(t, `modules.base16_dunst.input = "self:modules/base16_dunst"`, rep.Modules[0].Problem.Declared)
	require.Contains(t, rep.Modules[0].Problem.Hint, `modules.base16_dunst.input = "self:modules/base16-dunst"`)
}

func TestLoadBrokenModuleOptions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, writeFile(root, "modot.cue", "package modot\n\nmodules: broken: {}\n"))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", "broken"), 0o750))
	require.NoError(t, writeFile(filepath.Join(root, "modules", "broken"), "modot.cue", "package broken\n\noptions: {\n"))

	// A module that does not parse breaks the whole merge, not just itself.
	rep, err := configdash.Load(t.Context(), configdash.ScopeCodebase, root)
	require.Error(t, err)
	require.ErrorIs(t, err, rep.MergedErr)
	require.Contains(t, err.Error(), filepath.Join(root, "modules", "broken", "modot.cue"))
	require.Contains(t, err.Error(), "expected '}'")
}

func writeFile(root, name, body string) error {
	err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600)
	if err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}

	return nil
}

// A core: module has no fragment on disk. It is not a load failure: it is
// explained, and its options are whatever the config set.
func TestLoadCoreModule(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	config := "package modot\n\nmodules: icons: {input: \"core:base16-icons-linux\"}\n" +
		"\noptions: icons: input_dir: \"papirus:Papirus\"\n"
	require.NoError(t, writeFile(root, "modot.cue", config))

	rep, err := configdash.Load(t.Context(), configdash.ScopeHome, root)
	require.NoError(t, err, "a core module is not an error")

	byName := map[string]configdash.Module{}
	for _, m := range rep.Modules {
		byName[m.Name] = m
	}
	mod, ok := byName["icons"]
	require.True(t, ok, "icons module")
	require.Equal(t, "core", mod.Source)
	require.Equal(t, "base16-icons-linux", mod.Ref)
	require.False(t, mod.Unavailable, "nothing on disk to miss")
	require.NotNil(t, mod.Problem)
	require.Equal(t, "core", mod.Problem.Provider)
	require.Contains(t, mod.Problem.Summary, "no fragment on disk")
	require.Contains(t, mod.Problem.Hint, "core:base16-icons-linux")
	require.Empty(t, mod.File)
	require.Equal(t, map[string]any{"input_dir": "papirus:Papirus"}, mod.Merged)
}
