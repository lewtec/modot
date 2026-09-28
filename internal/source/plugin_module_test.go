package source

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/modot/internal/configcue"
	_ "github.com/lewtec/modot/internal/driver/env/native"
	_ "github.com/lewtec/modot/internal/module/prelude"
)

func TestCloneModuleConfigIsolatesNestedMaps(t *testing.T) {
	t.Parallel()
	orig := map[string]any{
		"items": map[string]any{"a": "src:a", "b": "src:b"},
		"tags":  []any{"x", map[string]any{"k": "v"}},
	}
	cloned := cloneModuleConfig(orig)
	items := cloned["items"].(map[string]any)
	items["a"] = "mutated"
	tags := cloned["tags"].([]any)
	tags[1].(map[string]any)["k"] = "mutated"

	origItems := orig["items"].(map[string]any)
	require.Equal(t, "src:a", origItems["a"], "original nested map mutated")
	origTags := orig["tags"].([]any)
	require.Equal(t, "v", origTags[1].(map[string]any)["k"], "original nested slice map mutated")
}

func TestCloneModuleConfigNilBecomesEmpty(t *testing.T) {
	t.Parallel()
	got := cloneModuleConfig(nil)
	require.NotNil(t, got)
	require.Empty(t, got)
	got["x"] = 1
}

func TestModuleScannerProcessMapReduceOrder(t *testing.T) {
	root := t.TempDir()
	modulesDir := filepath.Join(root, "modules")
	require.NoError(t, os.MkdirAll(modulesDir, 0o755))
	srcA := filepath.Join(root, "src-a")
	srcB := filepath.Join(root, "src-b")
	for _, dir := range []string{srcA, srcB} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	writeFile(t, filepath.Join(srcA, "a.txt"), "a")
	writeFile(t, filepath.Join(srcB, "b.txt"), "b")

	writeFile(t, filepath.Join(root, "modot.cue"), `package modot

modules: {
	zebra: {
		from: "core:place"
		config: {
			items: {
				"out-z": "`+srcB+`"
			}
		}
	}
	alpha: {
		from: "core:place"
		config: {
			items: {
				"out-a": "`+srcA+`"
			}
		}
	}
	noop: {
		enable: false
		from: "core:place"
	}
}
`)
	writeFile(t, filepath.Join(root, "modot.lock.json"), `{"dependencies":[]}`)

	g, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	t.Cleanup(func() {
		err := g.Wait()
		if !t.Failed() {
			assert.NoError(t, err, "group wait")
		}
	})

	cfg, err := configcue.LoadFiles(ctx, []string{filepath.Join(root, "modot.cue")})
	require.NoError(t, err, "load config")

	// core:place is rewritten onto file.home and the modules are disabled.
	plugin := NewModuleScannerPlugin(modulesDir, cfg, 100)
	out, err := plugin.Process(ctx, nil)
	require.NoError(t, err, "Process")
	require.Empty(t, out)
	mounts, err := cfg.FileMounts()
	require.NoError(t, err)
	require.Equal(t, []string{srcA}, mounts["home"]["out-a"].Srcs)
	require.Equal(t, []string{srcB}, mounts["home"]["out-z"].Srcs)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func moduleName(f File) string {
	if sf, ok := f.(ScopedFile); ok {
		return sf.ModuleName()
	}
	return ""
}
