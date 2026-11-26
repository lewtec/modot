package codebase

import (
	"bytes"
	"strings"
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/stretchr/testify/require"
)

func TestWriteTableKeepsOneFindingPerLine(t *testing.T) {
	t.Parallel()

	run := sarif.NewRun(*sarif.NewTool(sarif.NewDriver("deslop-please-rs")))
	run.AddResult(
		sarif.NewRuleResult("field-leak").
			WithLevel("error").
			WithMessage(sarif.NewTextMessage("`Native::args` is known outside `Native`\nsrc/fold.rs:728:1: `colcha::fold::fold_native`")).
			WithLocations([]*sarif.Location{
				sarif.NewLocation().WithPhysicalLocation(sarif.NewPhysicalLocation().
					WithArtifactLocation(sarif.NewArtifactLocation().WithUri("src/fold.rs")).
					WithRegion(sarif.NewRegion().WithStartLine(728))),
			}),
	)
	report := &sarif.Report{Version: "2.1.0", Runs: []*sarif.Run{run}}

	var out bytes.Buffer
	require.NoError(t, writeTable(&out, report))
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2, "header plus one row")
	require.Contains(t, lines[1], "src/fold.rs:728")
	require.Contains(t, lines[1], "`Native::args` is known outside `Native` src/fold.rs:728:1: `colcha::fold::fold_native`")
}
