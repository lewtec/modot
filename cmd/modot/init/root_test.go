package init

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestGenerateConfigAtomicWrite(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	configPath := filepath.Join(dir, "modot.cue")

	// Seed an existing config so --force-style overwrite cannot wipe it on failure
	// of a later stage. generateConfig always targets configPath via temp+rename.
	const prior = "// prior config must survive a failed write path\n"
	require.NoError(t, os.WriteFile(configPath, []byte(prior), 0o644))

	require.NoError(t, generateConfig(ctx, configPath), "generateConfig")

	got, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.NotEqual(t, prior, string(got), "config was not replaced with template output")
	require.Contains(t, string(got), "modules:")
	// Temp must not linger after success.
	_, err = os.Stat(configPath + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "temp file still present")
}

func TestGenerateConfigRemovesTempOnSuccess(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	configPath := filepath.Join(dir, "modot.cue")

	require.NoError(t, generateConfig(ctx, configPath), "generateConfig")
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.NotZero(t, info.Size(), "config is empty")
	_, err = os.Stat(configPath + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "temp file still present after success")
}
