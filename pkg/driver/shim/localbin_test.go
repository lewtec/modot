package shim_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/lucasew/workspaced/pkg/driver/prelude"
	"github.com/lucasew/workspaced/pkg/driver/shim"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestGenerateInLocalBin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("TERMUX_APP_PACKAGE", "")
	t.Setenv("WORKSPACED_IN_PROOT", "")
	t.Setenv("PREFIX", "")
	ctx := logging.NewWriterContext(t.Output())
	target := filepath.Join(home, "opt", "workspaced")

	shimPath, err := shim.GenerateInLocalBin(ctx, "workspaced", []string{target})
	require.NoError(t, err)

	wantPath := filepath.Join(home, ".local", "bin", "workspaced")
	require.Equal(t, wantPath, shimPath)

	info, err := os.Stat(shimPath)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "shim not executable: %o", info.Mode())

	content, err := os.ReadFile(shimPath)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(content), "#!"), "missing shebang: %q", content)
	require.Contains(t, string(content), target)
}

func TestGenerateInLocalBinValidation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := logging.NewWriterContext(t.Output())

	_, err := shim.GenerateInLocalBin(ctx, "", []string{"/bin/true"})
	require.ErrorIs(t, err, shim.ErrEmptyName)
	_, err = shim.GenerateInLocalBin(ctx, "workspaced", nil)
	require.ErrorIs(t, err, shim.ErrEmptyCommand)
	_, err = shim.GenerateInLocalBin(ctx, "../workspaced", []string{"/bin/true"})
	require.Error(t, err, "expected error for non-base shim name")
}

func TestLocalBinDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Ensure Termux rewrite does not apply in unit tests.
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("TERMUX_APP_PACKAGE", "")
	t.Setenv("WORKSPACED_IN_PROOT", "")
	t.Setenv("PREFIX", "")

	got, err := shim.LocalBinDir()
	require.NoError(t, err)
	want := filepath.Join(home, ".local", "bin")
	require.Equal(t, want, got)
}

// HOME under /home/<user> must not be re-prefixed when writing shims (the old
// /home/ string rewrite turned /home/user/... into /home/user/user/...).
func TestGenerateInLocalBinKeepsNormalHomePaths(t *testing.T) {
	// Use a path shaped like a real Linux home.
	root := t.TempDir()
	home := filepath.Join(root, "home", "user")
	require.NoError(t, os.MkdirAll(home, 0o755))
	t.Setenv("HOME", home)
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("TERMUX_APP_PACKAGE", "")
	t.Setenv("WORKSPACED_IN_PROOT", "")
	t.Setenv("PREFIX", "")

	target := filepath.Join(home, ".local", "share", "workspaced", "bin", "workspaced")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	ctx := logging.NewWriterContext(t.Output())
	shimPath, err := shim.GenerateInLocalBin(ctx, "workspaced", []string{target})
	require.NoError(t, err)
	content, err := os.ReadFile(shimPath)
	require.NoError(t, err)
	require.Contains(t, string(content), target)
	doubled := filepath.Join(home, "user")
	require.NotContains(t, string(content), doubled, "shim re-prefixed home (doubled user path)")
}
