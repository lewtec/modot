package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeBlankInput(t *testing.T) {
	t.Parallel()
	for _, decode := range []func(string, []byte) (any, error){
		func(name string, data []byte) (any, error) { return decodeShellcheck(name, data) },
		func(name string, data []byte) (any, error) { return decodeActionlint(name, data) },
	} {
		run, err := decode("tool", []byte(" \n"))
		require.NoError(t, err)
		require.Nil(t, run)
		run, err = decode("tool", []byte("[]"))
		require.NoError(t, err)
		require.Nil(t, run)
	}
}

func TestDecodeShellcheckRegionAndRule(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"file":"a.sh","line":4,"column":2,"endLine":5,"endColumn":8,"level":"error","code":2086,"message":"quote"}]`)
	run, err := decodeShellcheck("", raw)
	require.NoError(t, err)
	require.Equal(t, "shellcheck", run.Tool.Driver.Name)
	require.NotNil(t, run.Tool.Driver.InformationURI)
	require.Equal(t, shellcheckInfoURI, *run.Tool.Driver.InformationURI)
	require.Len(t, run.Results, 1)
	require.Equal(t, "SC2086", *run.Results[0].RuleID)
	loc := run.Results[0].Locations[0]
	require.Equal(t, "a.sh", *loc.PhysicalLocation.ArtifactLocation.URI)
	require.Equal(t, 4, *loc.PhysicalLocation.Region.StartLine)
	require.Equal(t, 2, *loc.PhysicalLocation.Region.StartColumn)
	require.Equal(t, 5, *loc.PhysicalLocation.Region.EndLine)
	require.Equal(t, 8, *loc.PhysicalLocation.Region.EndColumn)
}

func TestDecodeActionlintOmitsEndLine(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"message":"bad","filepath":".github/workflows/ci.yml","line":3,"column":1,"end_column":6,"kind":"syntax-check"}]`)
	run, err := decodeActionlint("actionlint", raw)
	require.NoError(t, err)
	require.Equal(t, "actionlint", run.Tool.Driver.Name)
	require.NotNil(t, run.Tool.Driver.InformationURI)
	require.Equal(t, actionlintInfoURI, *run.Tool.Driver.InformationURI)
	loc := run.Results[0].Locations[0]
	require.Equal(t, ".github/workflows/ci.yml", *loc.PhysicalLocation.ArtifactLocation.URI)
	require.Equal(t, 3, *loc.PhysicalLocation.Region.StartLine)
	require.Equal(t, 1, *loc.PhysicalLocation.Region.StartColumn)
	require.Nil(t, loc.PhysicalLocation.Region.EndLine)
	require.Equal(t, 6, *loc.PhysicalLocation.Region.EndColumn)
	require.Equal(t, "syntax-check", *run.Results[0].RuleID)
	require.Equal(t, "error", *run.Results[0].Level)
}
