package open

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/miseutil"
	_ "github.com/lewtec/modot/internal/driver/prelude"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestEnsureMiseWrapperAtomicWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := logging.NewWriterContext(t.Output())

	wrapperDir := filepath.Join(home, ".local", "bin")
	require.NoError(t, os.MkdirAll(wrapperDir, 0o755))
	wrapperPath := filepath.Join(wrapperDir, "mise")
	// Prior truncated/stale wrapper that must be fully replaced.
	require.NoError(t, os.WriteFile(wrapperPath, []byte("#!/bin/sh\necho prior\n"), 0o755))

	require.NoError(t, miseutil.EnsureLocalBinWrapper(ctx, "/fake/modot"), "EnsureLocalBinWrapper")

	_, err := os.Stat(wrapperPath + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "temp wrapper still present")
	content, err := os.ReadFile(wrapperPath)
	require.NoError(t, err)
	got := string(content)
	require.NotContains(t, got, "prior")
	require.Contains(t, got, "open lazy --home")
	require.Contains(t, got, "/fake/modot")
	info, err := os.Stat(wrapperPath)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "wrapper not executable: %o", info.Mode())
}
