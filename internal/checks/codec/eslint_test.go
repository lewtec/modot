package codec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAndConvert(t *testing.T) {
	input := []eslintResult{
		{
			FilePath: "/path/to/file.js",
			Messages: []eslintMessage{
				{RuleID: "no-unused-vars", Severity: 2, Message: "Unused variable", Line: 1, Column: 5, EndLine: 1, EndColumn: 10},
				{RuleID: "no-console", Severity: 1, Message: "Unexpected console statement", Line: 10, Column: 1, EndLine: 10, EndColumn: 10},
			},
		},
	}
	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)
	run, err := parseAndConvertESLint(jsonBytes)
	require.NoError(t, err)
	require.Len(t, run.Results, 2)
	require.Equal(t, "no-unused-vars", *run.Results[0].RuleID)
	require.Equal(t, "error", *run.Results[0].Level)
	require.Equal(t, "no-console", *run.Results[1].RuleID)
	require.Equal(t, "warning", *run.Results[1].Level)
}

func TestParseAndConvert_WithRawNewlineInString(t *testing.T) {
	raw := []byte(`[{"filePath":"/tmp/a.js","messages":[{"ruleId":"x","severity":2,"message":"line1
line2","line":1,"column":1,"endLine":1,"endColumn":2}]}]`)
	run, err := parseAndConvertESLint(raw)
	require.NoError(t, err)
	require.Len(t, run.Results, 1)
	require.NotNil(t, run.Results[0].Message.Text)
	require.Equal(t, "line1\nline2", *run.Results[0].Message.Text)
}

func TestParseAndConvert_WithNonJSONPrefix(t *testing.T) {
	raw := []byte(`note: running eslint
[{"filePath":"/tmp/b.js","messages":[{"ruleId":"y","severity":1,"message":"ok","line":2,"column":3,"endLine":2,"endColumn":4}]}]
done`)
	run, err := parseAndConvertESLint(raw)
	require.NoError(t, err)
	require.Len(t, run.Results, 1)
	require.NotNil(t, run.Results[0].RuleID)
	require.Equal(t, "y", *run.Results[0].RuleID)
}

func TestParseAndConvert_WithTruncatedJSONTail(t *testing.T) {
	raw := []byte(`[{"filePath":"/tmp/ok.js","messages":[{"ruleId":"ok","severity":1,"message":"first","line":1,"column":1,"endLine":1,"endColumn":2}]},{"filePath":"/tmp/broken.js","messages":[`)
	run, err := parseAndConvertESLint(raw)
	require.NoError(t, err)
	require.Len(t, run.Results, 1)
	require.Equal(t, "ok", *run.Results[0].RuleID)
}
