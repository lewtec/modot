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
	require.Equal(t, filepath.Join(root, "modules", "missing", "modot.cue"), problem.Looked)
	require.Equal(t, `modules.ghost.path = "missing"`, problem.Declared)
	require.Contains(t, problem.Hint, filepath.Join(root, "modules", "missing"))
	require.Contains(t, problem.Hint, "modules.ghost.enable = false")
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
