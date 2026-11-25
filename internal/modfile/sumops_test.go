package modfile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSumFileEnsureToolIsIdempotent(t *testing.T) {
	t.Parallel()

	sum := &SumFile{}
	sum.EnsureTool("gh", LockedTool{
		Ref:        "github:cli/cli",
		Version:    "2.89.0",
		DepName:    "cli/cli",
		Datasource: "github-releases",
	})

	lock, ok := sum.Tool("github:cli/cli")
	require.True(t, ok, "expected tool lock")
	require.Equal(t, "2.89.0", lock.Version)

	changed := sum.EnsureTool("gh", LockedTool{Ref: "github:cli/cli", Version: "2.89.0"})
	require.False(t, changed, "expected EnsureTool to be idempotent when passing minimal (no renovate fields)")
	require.Len(t, sum.Dependencies, 1)
	d := sum.Dependencies[0]
	require.Equal(t, "github-releases", d.Datasource)
	require.Equal(t, "cli/cli", d.DepName)
}

func TestUpsertToolRefreshesStaleCurrentValue(t *testing.T) {
	sum := &SumFile{}
	sum.EnsureTool("tirith", LockedTool{
		Ref:        "registry:tirith",
		Version:    "v0.3.1",
		DepName:    "sheeki03/tirith",
		Datasource: "github-releases",
	})

	changed := sum.EnsureTool("tirith", LockedTool{
		Ref:        "registry:tirith",
		Version:    "v0.3.1",
		DepName:    "sheeki03/tirith",
		Datasource: "github-releases",
		Versioning: "semver",
	})
	require.True(t, changed, "expected stale currentValue to be refreshed")

	dep := sum.Dependencies[0]
	require.Equal(t, "v0.3.1", dep.CurrentValue)
	require.Equal(t, "semver", dep.Versioning)
}
