package modfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRenovateDependenciesFromTools(t *testing.T) {
	t.Parallel()

	sources := map[string]LockedSource{
		"icons": {
			Provider: "github",
			Repo:     "PapirusDevelopmentTeam/papirus-icon-theme",
			Ref:      "v2026.03.01",
		},
		"theme": {
			Provider: "github",
			Repo:     "catppuccin/gtk",
			// LockHash writes the default branch into Ref; commit pin is in URL.
			Ref: "main",
			URL: "https://codeload.github.com/catppuccin/gtk/tar.gz/9aa0d1fabc1234",
		},
	}
	tools := map[string]LockedTool{
		"fd":   {Ref: "github:sharkdp/fd", Version: "v10.3.0", DepName: "sharkdp/fd", Datasource: "github-releases"},
		"fzf":  {Ref: "mise:fzf", Version: "0.50.0"},
		"bad1": {Ref: "", Version: "1.0.0"},
		"bad2": {Ref: "github:foo/bar", Version: ""},
	}

	got := BuildRenovateDependenciesFromLocks(sources, tools)
	require.Len(t, got, 4, "%#v", got)
	byName := map[string]RenovateDependency{}
	for _, dep := range got {
		byName[dep.DepName] = dep
	}
	sourceDep, ok := byName["PapirusDevelopmentTeam/papirus-icon-theme"]
	require.True(t, ok, "missing source dependency for papirus: %#v", got)
	require.Equal(t, "git-refs", sourceDep.Datasource)
	require.Equal(t, "https://github.com/PapirusDevelopmentTeam/papirus-icon-theme", sourceDep.PackageName)
	// Explicit non-SHA ref is the tracked git ref; no commit pin yet.
	require.Equal(t, "v2026.03.01", sourceDep.CurrentValue)
	require.Empty(t, sourceDep.CurrentDigest)
	toolDep, ok := byName["sharkdp/fd"]
	require.True(t, ok, "missing tool dependency for sharkdp/fd: %#v", got)
	require.Equal(t, "github-releases", toolDep.Datasource)
	require.Equal(t, "v10.3.0", toolDep.CurrentValue)

	// mise tool produces entry keyed by ref, no extra provider/name.
	var fzfDep *RenovateDependency
	for i := range got {
		if got[i].Ref == "mise:fzf" {
			fzfDep = &got[i]
			break
		}
	}
	require.NotNil(t, fzfDep, "expected fzf tool dep to be included for lock state")
	require.Empty(t, fzfDep.Datasource)

	themeDep, ok := byName["catppuccin/gtk"]
	require.True(t, ok, "missing source dependency for catppuccin/gtk: %#v", got)
	require.Equal(t, "git-refs", themeDep.Datasource)
	require.Equal(t, "main", themeDep.CurrentValue)
	require.Equal(t, "9aa0d1fabc1234", themeDep.CurrentDigest)
	require.Equal(t, "https://github.com/catppuccin/gtk", themeDep.PackageName)
}

func TestBuildRenovateDependenciesSkipsHeadWithoutBranch(t *testing.T) {
	t.Parallel()

	// HEAD alone (no resolved branch from LockHash) is not usable with git-refs.
	sources := map[string]LockedSource{
		"icons": {
			Provider: "github",
			Repo:     "PapirusDevelopmentTeam/papirus-icon-theme",
			Ref:      "HEAD",
		},
	}

	got := BuildRenovateDependenciesFromLocks(sources, nil)
	require.Len(t, got, 1, "%#v", got)
	require.Equal(t, "source", got[0].Kind)
	require.Empty(t, got[0].DepName)
}

func TestBuildRenovateDependenciesSkipsSHAOnlyWithoutBranch(t *testing.T) {
	t.Parallel()

	// Commit in URL but no named branch/tag => cannot satisfy git-refs.
	sources := map[string]LockedSource{
		"theme": {
			Provider: "github",
			Repo:     "catppuccin/gtk",
			URL:      "https://codeload.github.com/catppuccin/gtk/tar.gz/9aa0d1fabc1234",
		},
	}

	got := BuildRenovateDependenciesFromLocks(sources, nil)
	require.Len(t, got, 1)
	require.Empty(t, got[0].DepName)
	require.Empty(t, got[0].Datasource)
}

func TestMergeRenovateDependenciesPreservesUntouchedEntries(t *testing.T) {
	t.Parallel()

	existing := []RenovateDependency{
		{
			Kind:         "tool",
			Ref:          "github:sharkdp/fd",
			DepName:      "sharkdp/fd",
			CurrentValue: "v10.2.0",
			Datasource:   "github-releases",
		},
		{
			Kind:          "source",
			Ref:           "github:PapirusDevelopmentTeam/papirus-icon-theme",
			DepName:       "PapirusDevelopmentTeam/papirus-icon-theme",
			CurrentValue:  "master",
			CurrentDigest: "abc1234deadbeef",
			Datasource:    "git-refs",
			PackageName:   "https://github.com/PapirusDevelopmentTeam/papirus-icon-theme",
		},
	}

	generated := []RenovateDependency{
		{
			Kind:         "tool",
			Ref:          "github:sharkdp/fd",
			DepName:      "sharkdp/fd",
			CurrentValue: "v10.3.0",
			Datasource:   "github-releases",
		},
	}

	got := MergeRenovateDependencies(existing, generated)
	require.Len(t, got, 2)

	byRef := map[string]RenovateDependency{}
	for _, dep := range got {
		k := dep.Ref
		if k == "" {
			k = dep.DepName
		}
		byRef[k] = dep
	}
	require.Equal(t, "v10.3.0", byRef["github:sharkdp/fd"].CurrentValue)
	src := byRef["github:PapirusDevelopmentTeam/papirus-icon-theme"]
	require.Equal(t, "master", src.CurrentValue)
	require.Equal(t, "abc1234deadbeef", src.CurrentDigest)
}
