package materialyou

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/lucasew/workspaced/pkg/palette/api"
	"github.com/lucasew/workspaced/pkg/palette/palettetest"
	"github.com/stretchr/testify/require"
)

func loadGoldenPalette(t testing.TB, name string) *api.Palette {
	t.Helper()
	b, err := os.ReadFile(palettetest.Path(t, name))
	require.NoError(t, err, "read golden %s", name)
	var p api.Palette
	require.NoError(t, json.Unmarshal(b, &p), "parse golden %s", name)
	return &p
}

func TestMaterialYouFromTestdataSolid(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "solid_4285f4.png")
	ctx := logging.NewWriterContext(t.Output())
	d := &Driver{}

	dark, err := d.Extract(ctx, img, api.Options{Polarity: api.PolarityDark, ColorCount: 16})
	require.NoError(t, err)
	wantDark := loadGoldenPalette(t, "materialyou_dark16_4285f4.json")
	require.Equal(t, *wantDark, *dark)

	light, err := d.Extract(ctx, img, api.Options{Polarity: api.PolarityLight, ColorCount: 24})
	require.NoError(t, err)
	wantLight := loadGoldenPalette(t, "materialyou_light24_4285f4.json")
	require.Equal(t, *wantLight, *light)
}

func TestGenerateColorschemeSourceHex(t *testing.T) {
	t.Parallel()
	// Same dominant color as solid_4285f4.png / golden fixtures.
	scheme := GenerateColorscheme("#4285f4", nil)
	require.NotEmpty(t, scheme.Dark["surface"])
	require.NotEmpty(t, scheme.Light["primary"])
	// Extract maps surface → base00 for dark; golden locks that slot.
	want := loadGoldenPalette(t, "materialyou_dark16_4285f4.json")
	if got := scheme.Dark["surface"]; got != "#"+want.Base00 && got != want.Base00 {
		// colorsFor returns with or without # depending on Tone(); normalize
		g := got
		if len(g) == 7 && g[0] == '#' {
			g = g[1:]
		}
		require.Equal(t, want.Base00, g, "dark surface from golden base00")
	}
}

// bliss.jpg from ~/.dotfiles/assets/wallpapers — realistic multi-color source.
// MaxSamples must match the golden generator (CLI default 10000).
func TestMaterialYouFromTestdataBliss(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "bliss.jpg")
	ctx := logging.NewWriterContext(t.Output())
	d := &Driver{}
	opts := api.Options{Polarity: api.PolarityDark, ColorCount: 16, MaxSamples: 10000}

	dark, err := d.Extract(ctx, img, opts)
	require.NoError(t, err)
	wantDark := loadGoldenPalette(t, "materialyou_dark16_bliss.json")
	require.Equal(t, *wantDark, *dark)

	light, err := d.Extract(ctx, img, api.Options{Polarity: api.PolarityLight, ColorCount: 24, MaxSamples: 10000})
	require.NoError(t, err)
	wantLight := loadGoldenPalette(t, "materialyou_light24_bliss.json")
	require.Equal(t, *wantLight, *light)
}

func BenchmarkMaterialYouExtract(b *testing.B) {
	ctx := logging.NewWriterContext(b.Output())
	d := &Driver{}
	opts := api.Options{Polarity: api.PolarityDark, ColorCount: 16, MaxSamples: 10000}

	b.Run("gradient_256", func(b *testing.B) {
		img := palettetest.LoadImage(b, "gradient_256.png")
		b.ReportAllocs()
		for b.Loop() {
			_, err := d.Extract(ctx, img, opts)
			require.NoError(b, err)
		}
	})
	b.Run("bliss", func(b *testing.B) {
		img := palettetest.LoadImage(b, "bliss.jpg")
		b.ReportAllocs()
		for b.Loop() {
			_, err := d.Extract(ctx, img, opts)
			require.NoError(b, err)
		}
	})
}

func BenchmarkGenerateColorscheme(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		res := GenerateColorscheme("#4285f4", nil)
		var _ = res
		// named result (non-error return); var _ = to use without blank on call LHS
	}
}
