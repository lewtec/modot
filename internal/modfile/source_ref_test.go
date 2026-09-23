package modfile_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lucasew/workspaced/internal/modfile"
	_ "github.com/lucasew/workspaced/internal/modfile/sourceprovider/prelude"
)

func TestTryResolveSourceRefToPath(t *testing.T) {
	t.Parallel()

	mod := &modfile.ModFile{
		Sources: map[string]modfile.SourceConfig{
			"papirus": {
				Provider: "local",
				Path:     "/tmp/papirus-icon-theme-20250501",
			},
		},
	}

	got, ok, err := mod.TryResolveSourceRefToPath(t.Context(), "papirus:Papirus", "/home/lucasew/.dotfiles/modules")
	require.NoError(t, err)
	require.True(t, ok, "expected source ref to be resolved")
	want := filepath.Clean("/tmp/papirus-icon-theme-20250501/Papirus")
	require.Equal(t, want, filepath.Clean(got))
}

func TestTryResolveSourceRefToPathPlainPath(t *testing.T) {
	t.Parallel()

	mod := &modfile.ModFile{
		Sources: map[string]modfile.SourceConfig{},
	}

	input := "/tmp/papirus-icon-theme-20250501/Papirus"
	got, ok, err := mod.TryResolveSourceRefToPath(t.Context(), input, "/home/lucasew/.dotfiles/modules")
	require.NoError(t, err)
	require.False(t, ok, "did not expect plain path to be treated as source ref")
	require.Equal(t, input, got)
}

func TestTryResolveSourceRefToPathSelf(t *testing.T) {
	t.Parallel()

	// bare self input (from: "self") has Provider "self", empty Path.
	// "self:." and "self:subdir" must resolve relative to workspace root (Dir of modulesBaseDir).
	mod := &modfile.ModFile{
		Sources: map[string]modfile.SourceConfig{
			"skills_local_skills": {
				Provider: "self",
			},
		},
	}

	// modulesBaseDir points at <workspace>/modules
	modulesBase := "/home/user/dotfiles/modules"
	wsRoot := "/home/user/dotfiles"

	got, ok, err := mod.TryResolveSourceRefToPath(t.Context(), "skills_local_skills:.", modulesBase)
	require.NoError(t, err)
	require.True(t, ok, "expected self ref to resolve")
	require.Equal(t, wsRoot, got)

	got, ok, err = mod.TryResolveSourceRefToPath(t.Context(), "skills_local_skills:codex/skills", modulesBase)
	require.NoError(t, err)
	require.True(t, ok)
	want := filepath.Join(wsRoot, "codex/skills")
	require.Equal(t, want, got)
}

func TestTryResolveSourceRefToPathDirectSelf(t *testing.T) {
	t.Parallel()

	// direct self: form (no entry in Sources) should also work
	mod := &modfile.ModFile{Sources: map[string]modfile.SourceConfig{}}

	modulesBase := "/workspace/modules"
	got, ok, err := mod.TryResolveSourceRefToPath(t.Context(), "self:.", modulesBase)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "/workspace", got)
}
