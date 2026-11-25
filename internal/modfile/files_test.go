package modfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureLockFileCreatesLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sumPath, err := EnsureLockFile(t.Context(), root)
	require.NoError(t, err)

	_, err = os.Stat(sumPath)
	require.NoError(t, err, "sum file was not created")

	require.Equal(t, root, filepath.Dir(sumPath))
}
