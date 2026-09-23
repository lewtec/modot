package modfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspacePaths(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(t.TempDir())
	ws := NewWorkspace(root)
	require.Equal(t, root, ws.Root)
	require.Equal(t, filepath.Join(root, "workspaced.lock.json"), ws.SumPath())
	require.Equal(t, filepath.Join(root, "modules"), ws.ModulesBaseDir())
}
