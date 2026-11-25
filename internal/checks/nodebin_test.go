package checks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/checks"
	"github.com/stretchr/testify/require"
)

func TestNodeModuleBinRel(t *testing.T) {
	t.Parallel()
	got := checks.NodeModuleBinRel("prettier")
	want := filepath.Join("node_modules", ".bin", "prettier")
	require.Equal(t, want, got)
}

func TestRequireNodeModuleBin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := checks.RequireNodeModuleBin(dir, "eslint")
	require.ErrorIs(t, err, checks.ErrNotApplicable)

	binDir := filepath.Join(dir, "node_modules", ".bin")
	require.NoError(t, os.MkdirAll(binDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "eslint"), []byte("#!/bin/sh\n"), 0o644))
	require.NoError(t, checks.RequireNodeModuleBin(dir, "eslint"), "present bin")
}
