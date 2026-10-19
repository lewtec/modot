package modfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyReusableSourceLockKeepsVersioning(t *testing.T) {
	t.Parallel()

	entry := LockedSource{
		Provider: "github",
		Repo:     "jquery/jquery",
		Ref:      `tag:^v?(?<version>\d+\.\d+\.\d+)$`,
	}
	locked := LockedSource{
		Provider:   "github",
		Repo:       "jquery/jquery",
		Ref:        "4.0.0",
		Hash:       "4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
		URL:        "https://codeload.github.com/jquery/jquery/tar.gz/4f2fae08f23b54ce09322e62e73cce6161b8d3cb",
		Versioning: `regex:^v?(?<version>\d+\.\d+\.\d+)$`,
	}
	got := applyReusableSourceLock(entry, locked)
	require.Equal(t, "4.0.0", got.Ref)
	require.Equal(t, locked.Hash, got.Hash)
	require.Equal(t, locked.URL, got.URL)
	require.Equal(t, locked.Versioning, got.Versioning)
}
