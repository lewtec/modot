package genetic

import (
	"testing"

	"github.com/lucasew/workspaced/pkg/palette/api"
	"github.com/stretchr/testify/require"
)

func TestMapToPaletteCounts(t *testing.T) {
	t.Parallel()
	colors := make([]api.LAB, 24)
	for i := range colors {
		colors[i] = api.LAB{L: float64(i), A: 0, B: 0}
	}
	ind := Individual{colors: colors}

	p16 := mapToPalette(ind, 16)
	require.NotEmpty(t, p16.Base00)
	require.NotEmpty(t, p16.Base0F)
	require.Empty(t, p16.Base10)
	require.NotEqual(t, byte('#'), p16.Base00[0])

	p24 := mapToPalette(ind, 24)
	require.NotEmpty(t, p24.Base10)
	require.NotEmpty(t, p24.Base17)

	short := mapToPalette(Individual{colors: colors[:8]}, 16)
	require.Empty(t, short.Base00)
	require.Empty(t, short.Base0F)
}
