package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/modfile"
	"github.com/stretchr/testify/require"
)

func TestRehydrateLockedSource(t *testing.T) {
	t.Parallel()
	p := Provider{}
	dep := modfile.RenovateDependency{
		Kind:          "source",
		Ref:           "github:PapirusDevelopmentTeam/papirus-icon-theme",
		DepName:       "PapirusDevelopmentTeam/papirus-icon-theme",
		PackageName:   "https://github.com/PapirusDevelopmentTeam/papirus-icon-theme",
		CurrentValue:  "master",
		CurrentDigest: "702499f331aa9c38309e1af99de4021013916297",
		Datasource:    "git-refs",
	}
	lock, ok := p.RehydrateLockedSource(dep)
	require.True(t, ok, "expected github provider to own row")
	require.Equal(t, "github", lock.Provider, "identity: %#v", lock)
	require.Equal(t, "PapirusDevelopmentTeam/papirus-icon-theme", lock.Repo, "identity: %#v", lock)
	require.Equal(t, "master", lock.Ref)
	require.Equal(t, "702499f331aa9c38309e1af99de4021013916297", lock.Hash)
	wantURL := "https://codeload.github.com/PapirusDevelopmentTeam/papirus-icon-theme/tar.gz/702499f331aa9c38309e1af99de4021013916297"
	require.Equal(t, wantURL, lock.URL)
	require.True(t, p.LockReusable(lock), "expected reusable")
	_, ok = p.RehydrateLockedSource(modfile.RenovateDependency{Kind: "tool", Ref: "github:cli/cli"})
	require.False(t, ok, "must not own tool rows")
}

func TestLockMatchesDesired(t *testing.T) {
	t.Parallel()
	p := Provider{}
	locked := modfile.LockedSource{
		Provider: "github",
		Repo:     "PapirusDevelopmentTeam/papirus-icon-theme",
		Ref:      "master",
		Hash:     "702499f331aa9c38309e1af99de4021013916297",
		URL:      "https://codeload.github.com/PapirusDevelopmentTeam/papirus-icon-theme/tar.gz/702499f331aa9c38309e1af99de4021013916297",
	}
	require.True(t, p.LockMatchesDesired(modfile.LockedSource{}, locked), "empty desired")
	require.True(t, p.LockMatchesDesired(modfile.LockedSource{Ref: "HEAD"}, locked), "HEAD desired")
	require.True(t, p.LockMatchesDesired(modfile.LockedSource{Ref: "master"}, locked), "same branch")
	require.False(t, p.LockMatchesDesired(modfile.LockedSource{Ref: "develop"}, locked), "other branch")
	require.True(t, p.LockMatchesDesired(modfile.LockedSource{Ref: "702499f331aa9c38309e1af99de4021013916297"}, locked), "pin sha")
	require.False(t, p.LockReusable(modfile.LockedSource{Provider: "github", Hash: "abc", Ref: "702499f331aa9c38309e1af99de4021013916297"}), "sha-only tracking must not be reusable")

	const spec = `tag:^v?(?<version>\d+\.\d+\.\d+)$`
	tagged := modfile.LockedSource{Provider: "github", Ref: "4.0.0", Hash: "4f2fae08f23b54ce09322e62e73cce6161b8d3cb"}
	require.True(t, p.LockMatchesDesired(modfile.LockedSource{Ref: spec}, tagged), "tag pattern matches locked tag")
	require.False(t, p.LockMatchesDesired(modfile.LockedSource{Ref: spec}, locked), "tag pattern misses branch")
	require.False(t, p.LockMatchesDesired(modfile.LockedSource{Ref: spec}, modfile.LockedSource{Ref: spec, Hash: "abc"}), "unresolved pattern is not a match")
	require.False(t, p.LockMatchesDesired(modfile.LockedSource{Ref: "tag:("}, tagged), "invalid pattern does not match")
	require.False(t, p.LockReusable(modfile.LockedSource{Provider: "github", Hash: "abc", Ref: spec}), "unresolved tag pattern is not reusable")
	require.True(t, p.LockReusable(tagged), "resolved tag stays reusable")
}

func TestRehydrateGitTags(t *testing.T) {
	t.Parallel()
	p := Provider{}
	const versioning = `regex:^v?(?<version>\d+\.\d+\.\d+)$`
	dep := modfile.RenovateDependency{
		Kind:          "source",
		Ref:           "github:jquery/jquery",
		DepName:       "jquery/jquery",
		PackageName:   "https://github.com/jquery/jquery",
		CurrentValue:  "4.0.0",
		CurrentDigest: "4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
		Datasource:    "git-tags",
		Versioning:    versioning,
	}
	lock, ok := p.RehydrateLockedSource(dep)
	require.True(t, ok)
	require.Equal(t, "4.0.0", lock.Ref)
	require.Equal(t, versioning, lock.Versioning)
	require.Equal(t, "https://codeload.github.com/jquery/jquery/tar.gz/4f2fae08f23b54ce09322e62e73cce6161b8d3cb", lock.URL)
}

func TestUpsertSourceTagVersioning(t *testing.T) {
	t.Parallel()
	sum := &modfile.SumFile{}
	const versioning = `regex:^v?(?<version>\d+\.\d+\.\d+)$`
	entry := modfile.LockedSource{
		Provider:   "github",
		Repo:       "jquery/jquery",
		Ref:        "4.0.0",
		Hash:       "4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
		URL:        "https://codeload.github.com/jquery/jquery/tar.gz/4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
		Versioning: versioning,
	}
	require.True(t, sum.UpsertSource("jquery", entry))
	require.Equal(t, "git-tags", sum.Dependencies[0].Datasource)
	require.Equal(t, versioning, sum.Dependencies[0].Versioning)
	require.Equal(t, "4.0.0", sum.Dependencies[0].CurrentValue)
	require.False(t, sum.UpsertSource("jquery", entry), "same tag row is unchanged")

	branch := entry
	branch.Versioning = ""
	branch.Ref = "main"
	require.True(t, sum.UpsertSource("jquery", branch))
	require.Equal(t, "git-refs", sum.Dependencies[0].Datasource)
	require.Empty(t, sum.Dependencies[0].Versioning)
	require.Equal(t, "main", sum.Dependencies[0].CurrentValue)
}

func TestUpsertSourceIdempotentAfterReload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sumPath := filepath.Join(dir, "modot.lock.json")
	sum := &modfile.SumFile{}
	entry := modfile.LockedSource{
		Provider: "github",
		Repo:     "PapirusDevelopmentTeam/papirus-icon-theme",
		Ref:      "master",
		Hash:     "702499f331aa9c38309e1af99de4021013916297",
		URL:      "https://codeload.github.com/PapirusDevelopmentTeam/papirus-icon-theme/tar.gz/702499f331aa9c38309e1af99de4021013916297",
	}
	require.True(t, sum.EnsureSource("papirus", entry), "expected initial ensure to change")
	require.NoError(t, os.WriteFile(sumPath, []byte(`{
  "dependencies": [
    {
      "kind": "source",
      "ref": "github:PapirusDevelopmentTeam/papirus-icon-theme",
      "depName": "PapirusDevelopmentTeam/papirus-icon-theme",
      "packageName": "https://github.com/PapirusDevelopmentTeam/papirus-icon-theme",
      "currentValue": "master",
      "currentDigest": "702499f331aa9c38309e1af99de4021013916297",
      "datasource": "git-refs"
    }
  ]
}
`), 0o644))
	loaded, err := modfile.LoadSumFile(sumPath)
	require.NoError(t, err)
	lock, ok := loaded.FindSource("github:PapirusDevelopmentTeam/papirus-icon-theme")
	p := Provider{}
	require.True(t, ok, "reloaded lock not reusable: %#v", lock)
	require.True(t, p.LockReusable(lock), "reloaded lock not reusable: %#v", lock)
	_, ok = loaded.FindSource("PapirusDevelopmentTeam/papirus-icon-theme")
	require.True(t, ok, "missing lookup by repo key")
	reuse := entry
	reuse.Hash = lock.Hash
	reuse.URL = lock.URL
	reuse.Ref = lock.Ref
	require.False(t, loaded.UpsertSource("papirus", reuse), "expected idempotent upsert, deps=%#v", loaded.Dependencies)
}

func TestConfigureFromSpecAndModuleRef(t *testing.T) {
	t.Parallel()
	p := Provider{}
	cfg := modfile.SourceConfig{Provider: "github"}
	p.ConfigureFromSpec(&cfg, "owner/repo")
	require.Equal(t, "owner/repo", cfg.Repo, "cfg=%#v", cfg)
	require.Empty(t, cfg.Path, "cfg=%#v", cfg)
	fullRef, ver, handled, err := p.ResolveModuleRef(cfg, "subdir@v1")
	require.True(t, handled, "err=%v", err)
	require.NoError(t, err)
	require.Equal(t, "owner/repo/subdir", fullRef)
	require.Equal(t, "v1", ver)
}

func TestLockLookupKeys(t *testing.T) {
	t.Parallel()
	p := Provider{}
	keys := p.LockLookupKeys(modfile.LockedSource{Repo: "o/r"})
	require.Equal(t, []string{"github:o/r", "o/r"}, keys)
}

func TestCanPersistLock(t *testing.T) {
	t.Parallel()
	p := Provider{}
	lock := modfile.LockedSource{Provider: "github", Repo: "o/r"}
	require.False(t, p.CanPersistLock(modfile.RenovateDependency{Ref: "deadbeef"}, lock), "incomplete row must be rejected")
	require.True(t, p.CanPersistLock(modfile.RenovateDependency{DepName: "o/r", Datasource: "git-refs"}, lock), "complete row must persist")
}

func TestRefFromCodeloadTarballURL(t *testing.T) {
	t.Parallel()
	require.Equal(t, "abc1234", refFromCodeloadTarballURL("https://codeload.github.com/o/r/tar.gz/abc1234"))
	require.Empty(t, refFromCodeloadTarballURL("https://example.com/o/r/tar.gz/abc"))
}
