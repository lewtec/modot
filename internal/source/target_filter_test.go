package source

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucasew/workspaced/internal/configcue"
	_ "github.com/lucasew/workspaced/internal/module/prelude"
	_ "github.com/lucasew/workspaced/pkg/driver/env/native"
	"github.com/lucasew/workspaced/pkg/logging"
)

func TestStandardDotfilesKeepsCodebasePresetOnly(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "modules", "demo")
	writeFile(t, filepath.Join(modDir, "module.cue"), "package module\n\nmodule: { config: {} }\n")
	writeFile(t, filepath.Join(modDir, "home", ".bashrc"), "from-home\n")
	writeFile(t, filepath.Join(modDir, "codebase", ".gitignore"), "from-codebase\n")
	writeFile(t, filepath.Join(root, "workspaced.cue"), `package workspaced

workspaced: {
	modules: {
		demo: {
			enable: true
			input: "self"
			path: "modules/demo"
		}
	}
}
`)
	writeFile(t, filepath.Join(root, "workspaced.lock.json"), `{"dependencies":[]}`)

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	t.Cleanup(func() {
		err := g.Wait()
		if !t.Failed() {
			assert.NoError(t, err, "group wait")
		}
	})
	cfgCode, err := configcue.LoadForWorkspace(ctx, root)
	require.NoError(t, err, "load codebase config")
	cfgHome, err := configcue.LoadFiles(ctx, []string{filepath.Join(root, "workspaced.cue")})
	require.NoError(t, err, "load home config")

	t.Run("codebase mode", func(t *testing.T) {
		b, err := StandardDotfilesOptions{
			ConfigTreeTarget: root,
			ModulesDir:       filepath.Join(root, "modules"),
			ModulesCfg:       cfgCode,
		}.Builder(cfgCode)
		require.NoError(t, err)
		tree, err := b.Tree(ctx)
		require.NoError(t, err)
		got := fileTargets(tree.Files())
		want := []string{filepath.Join(root, ".gitignore")}
		require.Equal(t, want, got)
	})

	t.Run("home mode", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		b, err := StandardDotfilesOptions{
			ConfigTreeTarget: home,
			ModulesDir:       filepath.Join(root, "modules"),
			ModulesCfg:       cfgHome,
		}.Builder(cfgHome)
		require.NoError(t, err)
		tree, err := b.Tree(ctx)
		require.NoError(t, err)
		got := fileTargets(tree.Files())
		want := []string{filepath.Join(home, ".bashrc")}
		require.Equal(t, want, got)
	})
}

func fileTargets(files []File) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = filepath.Join(f.TargetBase(), f.RelPath())
	}
	return out
}
