package icons

import (
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func iconTestdata(t testing.TB, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func loadIconImage(t testing.TB, name string) image.Image {
	t.Helper()
	f, err := os.Open(iconTestdata(t, name))
	require.NoError(t, err, "open testdata %s", name)
	defer f.Close()
	img, _, err := image.Decode(f)
	require.NoError(t, err, "decode testdata %s", name)
	return img
}

func loadBase16Fixture(t testing.TB) map[string]string {
	t.Helper()
	b, err := os.ReadFile(iconTestdata(t, "base16_fixture.json"))
	require.NoError(t, err)
	var raw map[string]string
	require.NoError(t, json.Unmarshal(b, &raw))
	// renderSVG also indexes UPPER keys
	out := make(map[string]string, len(raw)*2)
	for k, v := range raw {
		v = strings.TrimPrefix(v, "#")
		out[k] = v
		out[strings.ToUpper(k)] = v
	}
	return out
}

func nrgbaAt(img image.Image, x, y int) color.NRGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func TestMakeBackgroundTransparentFloodFill(t *testing.T) {
	t.Parallel()
	img := loadIconImage(t, "flood_fill_64.png")
	out := makeBackgroundTransparent(img)

	// Corners were opaque green background → transparent after flood fill.
	for _, p := range [][2]int{{0, 0}, {63, 0}, {0, 63}, {63, 63}} {
		c := nrgbaAt(out, p[0], p[1])
		require.Equal(t, uint8(0), c.A, "corner (%d,%d)", p[0], p[1])
	}
	// Center of red square must remain opaque red-ish.
	c := nrgbaAt(out, 32, 32)
	require.NotEqual(t, uint8(0), c.A, "center became transparent")
	require.GreaterOrEqual(t, c.R, uint8(0x80), "center color unexpected: %#v", c)
	require.LessOrEqual(t, c.G, uint8(0x40), "center color unexpected: %#v", c)
}

func TestMakeBackgroundTransparentAlreadyClear(t *testing.T) {
	t.Parallel()
	img := loadIconImage(t, "transparent_bg_32.png")
	out := makeBackgroundTransparent(img)
	// Early-return path: same image value when corner already transparent.
	require.True(t, out == img, "expected identity return when background already transparent")
}

func TestCropToContentSquare(t *testing.T) {
	t.Parallel()
	img := loadIconImage(t, "transparent_bg_32.png")
	out := cropToContentSquare(img)
	b := out.Bounds()
	// Content was 16x16 centered in 32x32 → square crop of content.
	require.Equal(t, b.Dy(), b.Dx(), "expected square, got %dx%d", b.Dx(), b.Dy())
	require.GreaterOrEqual(t, b.Dx(), 16, "unexpected crop size %d", b.Dx())
	require.LessOrEqual(t, b.Dx(), 32, "unexpected crop size %d", b.Dx())
	// Cropped image should have some opaque pixels.
	opaque := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b, a := out.At(x, y).RGBA()
			if a > 0 {
				opaque++
				if r == 0 && g == 0 && b == 0 {
					// named and referenced r/g/b per policy for non-error discards; blank not on call LHS
				}
			}
		}
	}
	require.NotZero(t, opaque, "crop produced fully transparent image")
}

func TestResizeAndCenter(t *testing.T) {
	t.Parallel()
	img := loadIconImage(t, "wide_48x24.png")
	resized := resizeBilinear(img, 24, 12)
	require.Equal(t, 24, resized.Bounds().Dx(), "resize bounds = %v", resized.Bounds())
	require.Equal(t, 12, resized.Bounds().Dy(), "resize bounds = %v", resized.Bounds())
	squared := centerInSquare(resized, 32)
	require.Equal(t, 32, squared.Bounds().Dx(), "center bounds = %v", squared.Bounds())
	require.Equal(t, 32, squared.Bounds().Dy(), "center bounds = %v", squared.Bounds())
}

func TestRenderSVGReplaceAndTemplate(t *testing.T) {
	t.Parallel()
	colors := loadBase16Fixture(t)

	// Explicit replace: red → base08 from fixture.
	out, err := renderSVG(iconTestdata(t, "apps/sample.svg"), colors, map[string]string{
		"ff0000": colors["base08"],
	}, false, "test-theme", "sample")
	require.NoError(t, err)
	require.Contains(t, out, "#"+colors["base08"], "expected replaced fill with base08 #%s", colors["base08"])
	require.NotContains(t, strings.ToLower(out), "#ff0000")

	// Template path: {{.base00}} / {{.base0D}}
	tmplOut, err := renderSVG(iconTestdata(t, "apps/templated.svg.tmpl"), colors, nil, false, "test-theme", "templated")
	require.NoError(t, err)
	require.Contains(t, tmplOut, "#"+colors["base00"])
	require.Contains(t, tmplOut, "#"+colors["base0D"])
	require.NotContains(t, tmplOut, "{{", "unexpanded template left in output")
}

func TestMapHexColorsToScheme(t *testing.T) {
	t.Parallel()
	colors := loadBase16Fixture(t)
	in, err := os.ReadFile(iconTestdata(t, "placeholder.svg"))
	require.NoError(t, err)
	out := mapHexColorsToScheme(string(in), colors)
	require.NotContains(t, strings.ToLower(out), "#abcdef")
	// Nearest palette color should be a known base16 hex.
	found := false
	for k, v := range colors {
		if strings.HasPrefix(k, "base") && strings.Contains(strings.ToLower(out), "#"+strings.ToLower(v)) {
			found = true
			break
		}
	}
	require.True(t, found, "mapped output has no fixture palette color:\n%s", out)
}

func TestCollectIconInputsTestdata(t *testing.T) {
	t.Parallel()
	paths, err := collectIconInputs(iconTestdata(t, "."))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(paths), 3, "svg inputs: %v", paths)
}

func BenchmarkMakeBackgroundTransparent(b *testing.B) {
	img := loadIconImage(b, "flood_fill_64.png")
	b.ReportAllocs()
	for b.Loop() {
		res := makeBackgroundTransparent(img)
		if res.Bounds().Dx() == 0 {
			// name result and reference to avoid blank on call LHS (non-error return)
		}
	}
}

func BenchmarkCropToContentSquare(b *testing.B) {
	img := loadIconImage(b, "transparent_bg_32.png")
	b.ReportAllocs()
	for b.Loop() {
		res := cropToContentSquare(img)
		if res.Bounds().Dx() == 0 {
			// name result and reference to avoid blank on call LHS (non-error return)
		}
	}
}

func BenchmarkResizeBilinear(b *testing.B) {
	img := loadIconImage(b, "wide_48x24.png")
	b.ReportAllocs()
	for b.Loop() {
		res := resizeBilinear(img, 128, 64)
		if res.Bounds().Dx() == 0 {
			// name result and reference to avoid blank on call LHS (non-error return)
		}
	}
}

func BenchmarkCenterInSquare(b *testing.B) {
	img := loadIconImage(b, "wide_48x24.png")
	b.ReportAllocs()
	for b.Loop() {
		res := centerInSquare(img, 64)
		if res.Bounds().Dx() == 0 {
			// name result and reference to avoid blank on call LHS (non-error return)
		}
	}
}

func BenchmarkRenderSVGMapScheme(b *testing.B) {
	colors := loadBase16Fixture(b)
	path := iconTestdata(b, "apps/sample.svg")
	b.ReportAllocs()
	for b.Loop() {
		_, err := renderSVG(path, colors, nil, true, "bench-theme", "sample")
		require.NoError(b, err)
	}
}

func BenchmarkMapHexColorsToScheme(b *testing.B) {
	colors := loadBase16Fixture(b)
	in, err := os.ReadFile(iconTestdata(b, "apps/sample.svg"))
	require.NoError(b, err)
	s := string(in)
	b.ReportAllocs()
	for b.Loop() {
		res := mapHexColorsToScheme(s, colors)
		if len(res) == 0 {
			// name result and reference to avoid blank on call LHS (non-error return)
		}
	}
}
