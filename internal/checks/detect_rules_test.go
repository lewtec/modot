package checks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateDetectPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))
	res, err := EvaluateDetect(dir, map[string]DetectRule{
		"00-go": {Path: "go.mod", Enable: true},
	})
	require.NoError(t, err)
	require.True(t, res.Applicable)
	require.Equal(t, "00-go", res.RuleKey)
}

func TestEvaluateDetectFirstMatchWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))
	res, err := EvaluateDetect(dir, map[string]DetectRule{
		"00-deny":  {Path: "go.mod", Enable: false},
		"01-allow": {Path: "go.mod", Enable: true},
	})
	require.NoError(t, err)
	require.False(t, res.Applicable, "expected first deny")
	require.Equal(t, "00-deny", res.RuleKey)
}

func TestEvaluateDetectGlob(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.sh"), []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "x.sh"), []byte("#!/bin/sh\n"), 0o755))
	res, err := EvaluateDetect(dir, map[string]DetectRule{
		"00-sh": {Glob: "**/*.sh", Enable: true},
	})
	require.NoError(t, err)
	require.True(t, res.Applicable, "expected applicable, got %+v", res)
	files, err := CollectGlob(dir, "**/*.sh")
	require.NoError(t, err)
	require.Equal(t, []string{"ok.sh"}, files)
}

func TestEvaluateDetectEmpty(t *testing.T) {
	t.Parallel()
	res, err := EvaluateDetect(t.TempDir(), nil)
	require.NoError(t, err)
	require.False(t, res.Applicable, "empty detect should not apply")
}
