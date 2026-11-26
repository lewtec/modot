package checks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/configcue"
	_ "github.com/lewtec/modot/internal/driver/env/prelude"
	"github.com/lewtec/modot/internal/filespine"

	"github.com/stretchr/testify/require"
)

func TestDeslopPleaseRSLintPreset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cuePath := filepath.Join(dir, "modot.cue")
	require.NoError(t, os.WriteFile(cuePath, []byte("package modot\n"), 0o644))

	cfg, err := configcue.LoadFilesMode(t.Context(), []string{cuePath}, filespine.ModeCodebase)
	require.NoError(t, err)

	var lazy map[string]struct {
		Ref  string   `json:"ref"`
		Bins []string `json:"bins"`
	}
	require.NoError(t, cfg.Decode("lazy_tools", &lazy))
	require.Equal(t, "github:deterministic-machines/deslop-please", lazy["deslop_please_rs"].Ref)
	require.Equal(t, []string{"deslop-please"}, lazy["deslop_please_rs"].Bins)

	tools, err := LoadTools(cfg, "lint")
	require.NoError(t, err)
	var got Tool
	var found bool
	for _, tool := range tools {
		if tool.Name == "deslop-please-rs" {
			got = tool
			found = true
			break
		}
	}
	require.True(t, found, "codebase prelude should declare deslop-please-rs")
	require.True(t, got.Enable)
	require.Equal(t, []string{"deslop-please", "--sarif"}, got.Cmd)
	require.Equal(t, "sarif", got.Output)
	require.Equal(t, map[string]bool{"deslop_please_rs": true}, got.Needs)
	require.False(t, got.ArgsFromGlobs)

	empty := t.TempDir()
	res, err := EvaluateDetect(empty, got.Detect)
	require.NoError(t, err)
	require.False(t, res.Applicable, "no Cargo workspace should leave deslop-please-rs off")

	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "Cargo.toml"), []byte("[package]\n"), 0o644))
	res, err = EvaluateDetect(workspace, got.Detect)
	require.NoError(t, err)
	require.True(t, res.Applicable)
	require.Equal(t, "00-cargo-toml", res.RuleKey)

	nested := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(nested, "member"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "member", "Cargo.toml"), []byte("[package]\n"), 0o644))
	res, err = EvaluateDetect(nested, got.Detect)
	require.NoError(t, err)
	require.False(t, res.Applicable, "a nested manifest is not the workspace root")
}
