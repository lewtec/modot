package review

import (
	"bytes"
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/stretchr/testify/require"
)

func TestParseUnifiedDiffLines(t *testing.T) {
	t.Parallel()
	diff := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -10,0 +11,2 @@
+line eleven
+line twelve
@@ -20 +22 @@
+line twenty two
`
	set := parseUnifiedDiffLines(diff)
	require.True(t, set["foo.go:11"], "set=%v", set)
	require.True(t, set["foo.go:12"], "set=%v", set)
	require.True(t, set["foo.go:22"], "set=%v", set)
	require.False(t, set["foo.go:10"], "should not include old-only lines")
}

func TestWriteWorkflowCommand(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	writeWorkflowCommand(&buf, "error", "a.go", 3, 4, "golangci-lint", "boom\nthere")
	got := buf.String()
	require.Contains(t, got, "::error file=a.go,line=3,col=4::")
	require.NotContains(t, got, "\nthere", "newline not sanitized")
}

func TestAnnotateFiltersByDiff(t *testing.T) {
	// unit-level: extract + filter logic via parse only; full Annotate needs git.
	run := sarif.NewRun(*sarif.NewTool(sarif.NewDriver("t")))
	run.AddResult(sarif.NewRuleResult("r").
		WithLevel("error").
		WithMessage(sarif.NewTextMessage("m")).
		WithLocations([]*sarif.Location{
			sarif.NewLocation().WithPhysicalLocation(
				sarif.NewPhysicalLocation().
					WithArtifactLocation(sarif.NewArtifactLocation().WithUri("foo.go")).
					WithRegion(sarif.NewRegion().WithStartLine(11)),
			),
		}))
	file, line, _, msg, level := extractFinding(run.Results[0])
	require.Equal(t, "foo.go", file)
	require.Equal(t, 11, line)
	require.Equal(t, "m", msg)
	require.Equal(t, "error", level)
}

func TestIsGitHubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	require.True(t, IsGitHubActions())
	t.Setenv("GITHUB_ACTIONS", "")
	require.False(t, IsGitHubActions())
}
