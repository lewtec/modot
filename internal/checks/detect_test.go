package checks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasew/workspaced/internal/checks"
	"github.com/stretchr/testify/require"
)

func TestRequireFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644))
	require.NoError(t, checks.RequireFile(dir, "go.mod"), "present file")
	err := checks.RequireFile(dir, "missing.lock")
	require.ErrorIs(t, err, checks.ErrNotApplicable)
}
