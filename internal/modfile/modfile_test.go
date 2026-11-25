package modfile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveModuleSourceWithLockVersion(t *testing.T) {
	t.Parallel()

	mod := &ModFile{
		Sources: map[string]SourceConfig{},
	}
	got, err := mod.ResolveModuleSource("foo", "github:owner/repo/path@v1.2.3", "/tmp/modules", nil)
	require.NoError(t, err)
	require.Equal(t, "github", got.Provider)
	require.Equal(t, "owner/repo/path", got.Ref)
	require.Equal(t, "v1.2.3", got.Version)
}

func TestResolveModuleSourceLocalAlias(t *testing.T) {
	t.Parallel()

	mod := &ModFile{
		Sources: map[string]SourceConfig{
			"repo": {
				Provider: "local",
				Path:     "shared-modules",
			},
		},
	}

	got, err := mod.ResolveModuleSource("foo", "repo:base16-vim", "/home/user/dotfiles/modules", nil)
	require.NoError(t, err)

	want := filepath.Clean("/home/user/dotfiles/shared-modules/base16-vim")
	require.Equal(t, want, filepath.Clean(got.Ref))
}

func TestResolveModuleSourceCoreRejectsVersion(t *testing.T) {
	t.Parallel()

	mod := &ModFile{Sources: map[string]SourceConfig{}}
	_, err := mod.ResolveModuleSource("icons", "core:base16-icons-linux@v1", "/tmp/modules", nil)
	require.Error(t, err, "expected version validation error")
}

func TestResolveModuleSourceDefaultsToSelfModulePath(t *testing.T) {
	t.Parallel()

	mod := &ModFile{Sources: map[string]SourceConfig{}}

	got, err := mod.ResolveModuleSource("icons", "", "/tmp/modules", nil)
	require.NoError(t, err)
	require.Equal(t, "self", got.Provider)
	// Relative to workspace root; the self module provider joins with Dir(modulesBaseDir).
	require.Equal(t, "modules/icons", got.Ref)
}
