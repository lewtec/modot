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

func TestPatlintLintPreset(t *testing.T) {
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
	require.Equal(t, "registry:patlint", lazy["patlint"].Ref)
	require.Equal(t, []string{"patlint"}, lazy["patlint"].Bins)

	tools, err := LoadTools(cfg, "lint")
	require.NoError(t, err)
	var got Tool
	var found bool
	for _, tool := range tools {
		if tool.Name == "patlint" {
			got = tool
			found = true
			break
		}
	}
	require.True(t, found, "codebase prelude should declare patlint")
	require.True(t, got.Enable)
	require.Equal(t, []string{"patlint", "run", "--format", "sarif", "--", "."}, got.Cmd)
	require.Equal(t, "sarif", got.Output)
	require.Equal(t, map[string]bool{"patlint": true}, got.Needs)
	require.False(t, got.ArgsFromGlobs)

	empty := t.TempDir()
	res, err := EvaluateDetect(empty, got.Detect)
	require.NoError(t, err)
	require.False(t, res.Applicable, "no pack should leave patlint off")

	pack := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(pack, ".patlint"), 0o755))
	res, err = EvaluateDetect(pack, got.Detect)
	require.NoError(t, err)
	require.True(t, res.Applicable)
	require.Equal(t, "00-patlint-dir", res.RuleKey)

	scripts := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(scripts, "rules.rft"), []byte("(rule)\n"), 0o644))
	res, err = EvaluateDetect(scripts, got.Detect)
	require.NoError(t, err)
	require.True(t, res.Applicable)
	require.Equal(t, "10-rft", res.RuleKey)

	nested := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(nested, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "pkg", "rules.rft"), []byte("(rule)\n"), 0o644))
	res, err = EvaluateDetect(nested, got.Detect)
	require.NoError(t, err)
	require.False(t, res.Applicable, "nested .rft is outside the pack root")
}
