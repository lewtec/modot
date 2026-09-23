package modfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteSumFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "workspaced.lock.json")

	sum := &SumFile{}
	sum.EnsureSource("papirus", LockedSource{
		Provider: "github",
		Repo:     "PapirusDevelopmentTeam/papirus-icon-theme",
		URL:      "https://codeload.github.com/PapirusDevelopmentTeam/papirus-icon-theme/tar.gz/main",
		Hash:     "abc123",
		Ref:      "main",
	})
	sum.EnsureTool("fd", LockedTool{
		Ref:     "github:sharkdp/fd",
		Version: "v10.4.0",
	})
	err := writeSumFile(t.Context(), path, sum)
	require.NoError(t, err)

	got, err := LoadSumFile(path)
	require.NoError(t, err, "load written")
	// After persist, sources are keyed in deps by stable source ref
	// (e.g. "github:..." or by depName in fallback). The LockedSource.Ref
	// holds the pinned value.
	_, ok := got.FindSource("PapirusDevelopmentTeam/papirus-icon-theme")
	require.True(t, ok, "missing source lock entry: %#v", got.Dependencies)
	tool, ok := got.FindTool("github:sharkdp/fd")
	require.True(t, ok, "missing tool version in content: %#v", got.Dependencies)
	require.Equal(t, "v10.4.0", tool.Version, "missing tool version in content: %#v", got.Dependencies)
	_, err = os.Stat(path + ".tmp")
	require.ErrorIs(t, err, os.ErrNotExist, "temp file should be gone after successful write")
}

func TestWriteSumFileRemovesTempOnRenameFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Destination is a directory so rename(tmp → path) fails with EISDIR.
	path := filepath.Join(dir, "workspaced.lock.json")
	require.NoError(t, os.Mkdir(path, 0o755))

	sum := &SumFile{}
	sum.EnsureTool("fd", LockedTool{Ref: "github:sharkdp/fd", Version: "v10.4.0"})
	err := writeSumFile(t.Context(), path, sum)
	require.Error(t, err, "expected rename error when destination is a directory")
	_, err = os.Stat(path + ".tmp")
	require.ErrorIs(t, err, os.ErrNotExist, "temp file should be cleaned up after rename failure")
}

func TestBuildSourceLockEntries(t *testing.T) {
	t.Parallel()

	mod := &ModFile{
		Sources: map[string]SourceConfig{
			"papirus": {Provider: "github", Repo: "PapirusDevelopmentTeam/papirus-icon-theme"},
		},
	}

	got := BuildSourceLockEntries(mod)
	entry, ok := got["papirus"]
	require.True(t, ok, "expected papirus source lock")
	require.Equal(t, "github", entry.Provider)
	require.Equal(t, "PapirusDevelopmentTeam/papirus-icon-theme", entry.Repo)
}
