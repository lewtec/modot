package cmdarg

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/workspaced/pkg/palette/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintFormat(t *testing.T) {
	def := cmd.ParseOK[struct {
		F cmd.EnumArg[LintFormat] `long:"format" default:"table"`
	}](t)
	assert.Equal(t, LintTable, def.F.Value())

	got := cmd.ParseOK[struct {
		F cmd.EnumArg[LintFormat] `long:"format"`
	}](t, "--format", "sarif")
	assert.Equal(t, LintSARIF, got.F.Value())
}

func TestPolarityAPI(t *testing.T) {
	got := cmd.ParseOK[struct {
		P cmd.EnumArg[Polarity] `long:"polarity" default:"any"`
	}](t, "--polarity", "dark")
	assert.Equal(t, api.PolarityDark, got.P.Value().API())
}

func TestUrgencyRejectsUnknown(t *testing.T) {
	var c cmd.Command[struct {
		U cmd.EnumArg[Urgency] `long:"urgency"`
	}]
	err := c.Parse("--urgency", "loud")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
}

func TestLayersFormatUsage(t *testing.T) {
	text, err := cmd.Usage[struct {
		F cmd.EnumArg[LayersFormat] `long:"format" help:"output format" default:"paths"`
	}]("layers")
	require.NoError(t, err)
	assert.Contains(t, text, "choices: paths, table")
}
