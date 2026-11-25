package api

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeHex(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"#aabbcc", "aabbcc"},
		{"aabbcc", "aabbcc"},
		{"#", ""},
	}
	for _, tc := range cases {
		got := NormalizeHex(tc.in)
		assert.Equal(t, tc.want, got, "NormalizeHex(%q)", tc.in)
	}
}

func TestPaletteFromHexes(t *testing.T) {
	t.Parallel()
	p := PaletteFromHexes([]string{"#112233", "445566"})
	require.Equal(t, "112233", p.Base00)
	require.Equal(t, "445566", p.Base01)
	require.Empty(t, p.Base02)
	require.Empty(t, p.Base10)

	full := make([]string, 24)
	for i := range full {
		full[i] = "000000"
	}
	full[0] = "#ffffff"
	full[16] = "#abcdef"
	p24 := PaletteFromHexes(full)
	require.Equal(t, "ffffff", p24.Base00)
	require.Equal(t, "abcdef", p24.Base10)
	require.Equal(t, "000000", p24.Base17)

	overflow := make([]string, 0, 25)
	overflow = append(overflow, full...)
	overflow = append(overflow, "deadbeef")
	pExtra := PaletteFromHexes(overflow)
	require.Equal(t, "000000", pExtra.Base17)
}

func TestToHex(t *testing.T) {
	t.Parallel()
	got := ToHex(color.RGBA{R: 0x42, G: 0x85, B: 0xf4, A: 0xff})
	require.Equal(t, "4285f4", got)
}

func TestRGBToLABRoundTripLightness(t *testing.T) {
	t.Parallel()
	c := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	lab := RGBToLAB(c)
	require.GreaterOrEqual(t, lab.L, 99.0)
	require.LessOrEqual(t, lab.L, 100.1)
	back := LABToRGB(lab)
	require.GreaterOrEqual(t, back.R, uint8(250))
	require.GreaterOrEqual(t, back.G, uint8(250))
	require.GreaterOrEqual(t, back.B, uint8(250))
}
