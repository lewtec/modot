package materialyou

import (
	"image"
	"image/color"
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/palette/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterialYouDriver(t *testing.T) {
	t.Parallel()
	d := &Driver{}
	assert.Equal(t, "materialyou", d.Name())

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{66, 133, 244, 255}) // #4285F4

	ctx := logging.NewWriterContext(t.Output())

	pal, err := d.Extract(ctx, img, api.Options{Polarity: api.PolarityDark, ColorCount: 16})
	require.NoError(t, err)

	assert.NotEmpty(t, pal.Base00, "expected Base00 to be set")
	assert.Empty(t, pal.Base10, "expected Base10 to be empty for ColorCount 16")
	assert.False(t, len(pal.Base00) != 6 || pal.Base00[0] == '#', "expected 6-digit hex without '#', got %q", pal.Base00)

	pal24, err := d.Extract(ctx, img, api.Options{Polarity: api.PolarityLight, ColorCount: 24})
	require.NoError(t, err)
	assert.NotEmpty(t, pal24.Base10, "expected Base10 to be set for ColorCount 24")
}

// Golden match against solid_4285f4.png + JSON fixtures lives in testdata_test.go.
