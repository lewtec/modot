package checks

import (
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/stretchr/testify/require"
)

func TestBundleRunsSkipsNilAndPreservesOrder(t *testing.T) {
	r1 := sarif.NewRun(*sarif.NewTool(sarif.NewDriver("a")))
	r2 := sarif.NewRun(*sarif.NewTool(sarif.NewDriver("b")))
	report, err := BundleRuns(r1, nil, r2)
	require.NoError(t, err)
	require.Len(t, report.Runs, 2)
	require.Equal(t, "a", report.Runs[0].Tool.Driver.Name)
	require.Equal(t, "b", report.Runs[1].Tool.Driver.Name)
}

func TestBundleRunsEmpty(t *testing.T) {
	report, err := BundleRuns()
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Empty(t, report.Runs)
}
