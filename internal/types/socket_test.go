package types_test

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/types"
	"github.com/stretchr/testify/require"
)

func TestDaemonSocketPathUsesXDGRuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/tmp/runtime-test")
	got := types.DaemonSocketPath()
	want := filepath.Join("/tmp/runtime-test", "modot.sock")
	require.Equal(t, want, got)
}

func TestDaemonSocketPathFallsBackWithoutXDG(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	got := types.DaemonSocketPath()
	require.Equal(t, "modot.sock", filepath.Base(got))
	require.False(t, filepath.Dir(got) == "" || filepath.Dir(got) == ".", "unexpected dir in %q", got)
}
