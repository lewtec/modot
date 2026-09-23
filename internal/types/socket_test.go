package types_test

import (
	"path/filepath"
	"testing"

	"github.com/lucasew/workspaced/internal/types"
	"github.com/stretchr/testify/require"
)

func TestDaemonSocketPathUsesXDGRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/tmp/runtime-test")
	got := types.DaemonSocketPath()
	want := filepath.Join("/tmp/runtime-test", "workspaced.sock")
	require.Equal(t, want, got)
}

func TestDaemonSocketPathFallsBackWithoutXDG(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	got := types.DaemonSocketPath()
	require.Equal(t, "workspaced.sock", filepath.Base(got))
	require.False(t, filepath.Dir(got) == "" || filepath.Dir(got) == ".", "unexpected dir in %q", got)
}
