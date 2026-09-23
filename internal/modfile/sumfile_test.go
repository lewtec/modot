package modfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSumFileRequiresSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumPath := filepath.Join(dir, "workspaced.lock.json")
	content, err := json.Marshal(map[string]any{
		"modules": map[string]any{
			"foo": map[string]any{"version": "v1.0.0"},
		},
	})
	require.NoError(t, err, "marshal")
	require.NoError(t, os.WriteFile(sumPath, content, 0644), "write sum")

	got, err := LoadSumFile(sumPath)
	require.NoError(t, err)
	require.Empty(t, got.Dependencies)
}

func TestLoadSumFileRequiresSourceProvider(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumPath := filepath.Join(dir, "workspaced.lock.json")
	content, err := json.Marshal(map[string]any{
		"sources": map[string]any{
			"papirus": map[string]any{"path": "/tmp/papirus"},
		},
	})
	require.NoError(t, err, "marshal")
	require.NoError(t, os.WriteFile(sumPath, content, 0644), "write sum")

	_, loadErr := LoadSumFile(sumPath)
	// sources top-level is no longer processed (leftovers removed); load succeeds with empty deps.
	require.NoError(t, loadErr, "legacy sources shape")
}

func TestLoadSumFileRequiresSourceHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumPath := filepath.Join(dir, "workspaced.lock.json")
	content, err := json.Marshal(map[string]any{
		"sources": map[string]any{
			"papirus": map[string]any{
				"provider": "github",
				"repo":     "PapirusDevelopmentTeam/papirus-icon-theme",
				"url":      "https://codeload.github.com/PapirusDevelopmentTeam/papirus-icon-theme/tar.gz/main",
			},
		},
	})
	require.NoError(t, err, "marshal")
	require.NoError(t, os.WriteFile(sumPath, content, 0644), "write sum")

	_, loadErr := LoadSumFile(sumPath)
	// sources top-level is no longer processed (leftovers removed).
	require.NoError(t, loadErr, "legacy sources shape")
}

func TestLoadSumFileMissingIsEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumPath := filepath.Join(dir, "missing.lock.json")
	got, err := LoadSumFile(sumPath)
	require.NoError(t, err)
	require.Empty(t, got.Dependencies)
}

func TestGenericSourceLockFallbackWithoutProvider(t *testing.T) {
	t.Parallel()

	locked := LockedSource{Ref: "v1", Hash: "abc"}
	require.True(t, sourceLockReusable(locked), "generic reusable requires hash only")
	require.True(t, sourceLockMatchesDesired(LockedSource{}, locked), "empty desired matches")
	require.True(t, sourceLockMatchesDesired(LockedSource{Ref: "v1"}, locked), "same ref matches")
	require.False(t, sourceLockMatchesDesired(LockedSource{Ref: "v2"}, locked), "different ref must not match")
}

func TestLoadSumFileToolLockUsesCurrentValueOverVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sumPath := filepath.Join(dir, "workspaced.lock.json")
	require.NoError(t, os.WriteFile(sumPath, []byte(`{
  "dependencies": [
    {
      "kind": "tool",
      "name": "ripgrep",
      "ref": "github:burntsushi/ripgrep",
      "version": "15.1.0",
      "currentValue": "14.1.1"
    }
  ]
}
`), 0644), "write sum")

	got, err := LoadSumFile(sumPath)
	require.NoError(t, err, "load sum")
	lock, ok := got.Tool("github:burntsushi/ripgrep")
	require.True(t, ok, "missing ripgrep lock")
	require.Equal(t, "14.1.1", lock.Version)
}
