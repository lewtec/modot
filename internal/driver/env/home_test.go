package env_test

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/driver/env"
	"github.com/stretchr/testify/require"
)

func TestNormalizeHomeTermuxChroot(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.3")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("HOME", "/home")

	got := env.NormalizeHome("/home")
	want := "/data/data/com.termux/files/home"
	require.Equal(t, want, got)
}

func TestNormalizeHomeTermuxAlreadyAbsolute(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.3")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	realHome := "/data/data/com.termux/files/home"
	require.Equal(t, realHome, env.NormalizeHome(realHome))
}

func TestNormalizeHomeNonTermuxLeavesHome(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("TERMUX_APP_PACKAGE", "")
	t.Setenv("MODOT_IN_PROOT", "")
	t.Setenv("PREFIX", "/usr")
	require.Equal(t, "/home", env.NormalizeHome("/home"))
}

func TestResolveHomeDirTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.3")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("HOME", "/home")

	got, err := env.ResolveHomeDir()
	require.NoError(t, err)
	want := filepath.Join("/data/data/com.termux/files", "home")
	require.Equal(t, want, got)
}

func TestIsTermuxLike(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("TERMUX_APP_PACKAGE", "")
	t.Setenv("MODOT_IN_PROOT", "")
	t.Setenv("PREFIX", "/usr")
	require.False(t, env.IsTermuxLike(), "expected false without markers")

	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	require.True(t, env.IsTermuxLike(), "expected true with Termux PREFIX")
}
