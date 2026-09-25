package bash_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/modot/internal/driver/prelude"
	"github.com/lewtec/modot/internal/driver/shim"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateWritesViaTempRename(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	path := filepath.Join(dir, "tool")

	prior := "#!/bin/sh\necho prior\n"
	err := os.WriteFile(path, []byte(prior), 0o755)
	require.NoError(t, err)

	target := filepath.Join(dir, "real-bin")
	require.NoError(t, shim.Generate(ctx, path, []string{target}))

	_, err = os.Stat(path + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "temp path still present after success")

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	got := string(content)
	require.NotContains(t, got, "prior")
	require.Contains(t, got, target)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "shim not executable: %o", info.Mode())
}

func TestGeneratePreservesExistingOnTempWriteFailure(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	path := filepath.Join(dir, "tool")

	prior := "#!/bin/sh\necho keep-me\n"
	err := os.WriteFile(path, []byte(prior), 0o755)
	require.NoError(t, err)

	// Make the directory non-writable so creating path+".tmp" fails while the
	// existing final path remains readable. Restore perms in cleanup so TempDir
	// removal succeeds.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() {
		assert.NoError(t, os.Chmod(dir, 0o755), "chmod restore")
	})

	err = shim.Generate(ctx, path, []string{"/bin/true"})
	require.Error(t, err, "expected Generate to fail when temp cannot be written")

	// Restore write so we can read the preserved final path.
	require.NoError(t, os.Chmod(dir, 0o755))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, prior, string(content))
	_, err = os.Stat(path + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "temp path left behind after failure")
}
