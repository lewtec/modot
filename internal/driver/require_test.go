package driver_test

import (
	"testing"

	"github.com/lewtec/modot/internal/executil"
	"github.com/lewtec/modot/internal/driver"
	"github.com/stretchr/testify/require"
)

func TestRequireEnv(t *testing.T) {
	ctx := executil.WithEnv(t.Context(), []string{"FOO=1"})
	require.NoError(t, driver.RequireEnv(ctx, "FOO"))
	require.ErrorIs(t, driver.RequireEnv(ctx, "MISSING"), driver.ErrIncompatible)
}

func TestRequireAnyEnv(t *testing.T) {
	ctx := executil.WithEnv(t.Context(), []string{"WAYLAND_DISPLAY=wayland-0"})
	require.NoError(t, driver.RequireAnyEnv(ctx, "DISPLAY", "WAYLAND_DISPLAY"))
	require.ErrorIs(t, driver.RequireAnyEnv(ctx, "DISPLAY", "OTHER"), driver.ErrIncompatible)
}

func TestRequireTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	require.ErrorIs(t, driver.RequireTermux(), driver.ErrIncompatible)
	t.Setenv("TERMUX_VERSION", "0.118")
	require.NoError(t, driver.RequireTermux())
	require.True(t, driver.IsTermux(), "IsTermux false")
}
