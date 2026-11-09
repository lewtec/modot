package configdash_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/modot/internal/configdash"
	_ "github.com/lewtec/modot/internal/driver/prelude"
	"github.com/stretchr/testify/require"
)

// filesFixtureRoot is the workspace whose file tree the tests compose.
func filesFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller")

	return filepath.Join(filepath.Dir(file), "testdata", "files")
}

// The files page must show the files apply would place, not a guess: the tree
// comes from the same pipeline, so a module's file and the config tree's
// template both have to show up.
func TestLoadFilesFixture(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "LoadFiles")
	require.Equal(t, "codebase", files.Mode)
	require.Equal(t, []string{"codebase"}, files.Profiles)
	// The page abbreviates the home directory to ~, so a workspace under it
	// reads by its short form.
	home, homeErr := os.UserHomeDir()
	require.NoError(t, homeErr, "UserHomeDir")
	wantTarget := root
	rel, relErr := filepath.Rel(home, root)
	if relErr == nil && !strings.HasPrefix(rel, "..") {
		wantTarget = "~/" + filepath.ToSlash(rel)
	}
	require.Equal(t, wantTarget, files.TargetBase)

	byPath := map[string]configdash.MaterialFile{}
	for _, file := range files.Files {
		byPath[file.Path] = file
	}
	require.Contains(t, byPath, "from-module.txt")
	require.Contains(t, byPath, "from-tree.txt")

	// A file a module ships is credited to the module, so the page answers
	// "why is this here" without opening the module.
	require.Equal(t, "theme", byPath["from-module.txt"].Source)
	require.Equal(t, "static", byPath["from-module.txt"].Kind)
	require.Equal(t, "0644", byPath["from-module.txt"].Mode)

	// Groups are how the page lays the tree out; every file is in one.
	grouped := 0
	for _, group := range files.Groups {
		grouped += len(group.Files)
	}
	require.Equal(t, files.MaterialCount(), grouped)
}

// A scope with no config still answers: the page shows the error next to what
// it knows, instead of failing the request.
func TestLoadFilesNoConfig(t *testing.T) {
	t.Parallel()

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, t.TempDir())
	require.ErrorIs(t, err, configdash.ErrNoConfig)
	require.ErrorIs(t, files.Err, configdash.ErrNoConfig)
	require.Zero(t, files.MaterialCount())
}

// A tree that composes but places nothing is not an error: the scope's config
// simply declares no file entries for its mode.
func TestLoadFilesEmptyTree(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, writeFile(root, "modot.cue", "package modot\n\nmodules: {}\n"))

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "an empty tree is valid")
	require.Zero(t, files.MaterialCount())
	require.Empty(t, files.Groups)
}

// Home places files under the user's directory, because that is what home
// apply does; a page that said otherwise would mislead.
func TestLoadFilesHomeTarget(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	home, err := os.UserHomeDir()
	require.NoError(t, err, "UserHomeDir")
	if home == "" {
		t.Skip("no home directory")
	}

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeHome, root)
	require.NoError(t, err, "LoadFiles")
	require.Equal(t, "~", files.TargetBase)
}

// Groups collect the files under their directory, deepest first, so a module
// tree is not scattered across the page.
func TestLoadFilesGroups(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "LoadFiles")

	seen := map[string]bool{}
	for _, group := range files.Groups {
		require.False(t, seen[group.Dir], "group %q appears twice", group.Dir)
		seen[group.Dir] = true
		for _, file := range group.Files {
			require.Equal(t, group.Dir, filepath.Join(group.Root, file.Dir))
			require.False(t, strings.HasPrefix(file.Target, "/"), "target %q is not absolute", file.Target)
		}
	}
}
