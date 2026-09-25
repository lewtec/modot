package local

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/module"
)

func TestResolvePresetBases(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	initRepo(t, root)
	modulesDir := filepath.Join(root, "modules")
	modPath := filepath.Join(modulesDir, "demo")
	writeModuleTree(t, modPath, map[string]string{
		"home/.bashrc":         "home\n",
		"codebase/.gitignore":  "repo\n",
		"etc/nginx/nginx.conf": "nginx\n",
		"README.md":            "docs\n",
		"modot.cue":            "package module\n\nmodule: { config: {} }\n",
	})

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	got, err := (&Provider{}).Resolve(logging.NewWriterContext(t.Output()), module.ResolveRequest{
		Ref:            modPath,
		ModulesBaseDir: modulesDir,
	})
	require.NoError(t, err, "Resolve")

	want := []module.ResolvedFile{
		{RelPath: ".bashrc", TargetBase: home},
	}
	opts := []cmp.Option{
		cmpopts.IgnoreFields(module.ResolvedFile{}, "Mode", "Info", "AbsPath", "Symlink"),
		cmpopts.SortSlices(func(a, b module.ResolvedFile) bool {
			if a.TargetBase != b.TargetBase {
				return a.TargetBase < b.TargetBase
			}
			return a.RelPath < b.RelPath
		}),
	}
	diff := cmp.Diff(want, got.Files, opts...)
	require.Empty(t, diff)
}

func TestResolvePresetBaseUsesContextPrefix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx := cmdarg.WithPrefix(t.Context(), dir)
	for _, preset := range []string{"home", "codebase", "etc"} {
		got, err := resolvePresetBase(ctx, preset, "/ws/modules")
		require.NoError(t, err)
		require.Equal(t, dir, got, preset)
	}
}

func TestResolveUnknownPreset(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	initRepo(t, root)
	modulesDir := filepath.Join(root, "modules")
	modPath := filepath.Join(modulesDir, "demo")
	writeModuleTree(t, modPath, map[string]string{
		"nope/file.txt": "x\n",
		"modot.cue":     "package module\n\nmodule: { config: {} }\n",
	})

	_, err := (&Provider{}).Resolve(logging.NewWriterContext(t.Output()), module.ResolveRequest{
		Ref:            modPath,
		ModulesBaseDir: modulesDir,
	})
	require.ErrorIs(t, err, ErrUnknownPreset)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", dir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestResolvePresetBase(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name           string
		preset         string
		modulesBaseDir string
		want           string
		wantErr        error
	}{
		{name: "home", preset: "home", modulesBaseDir: "/ws/modules", want: home},
		{name: "codebase", preset: "codebase", modulesBaseDir: "/ws/modules", want: "/ws"},
		{name: "etc", preset: "etc", modulesBaseDir: "/ws/modules", want: "/"},
		{name: "bin", preset: "bin", modulesBaseDir: "/ws/modules", want: "/"},
		{name: "root", preset: "root", modulesBaseDir: "/ws/modules", want: "/"},
		{name: "unknown", preset: "nope", modulesBaseDir: "/ws/modules", wantErr: ErrUnknownPreset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolvePresetBase(t.Context(), tt.preset, tt.modulesBaseDir)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.want, got)
		})
	}
}

func writeModuleTree(t *testing.T, modPath string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(modPath, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
}
