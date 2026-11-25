package api

import (
	"testing"

	"github.com/lewtec/modot/internal/palette/palettetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSampleImageBlocks64(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "blocks_64.png")
	// One transparent pixel at (0,0); four opaque quadrant colors.
	colors := SampleImage(img, 0)
	require.Len(t, colors, 4)

	want := map[uint32]bool{
		0xe53935: true,
		0x43a047: true,
		0x1e88e5: true,
		0xfbc02d: true,
	}
	for _, c := range colors {
		key := uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
		assert.Contains(t, want, key, "unexpected color #%02x%02x%02x", c.R, c.G, c.B)
		delete(want, key)
	}
	assert.Empty(t, want)
}

func TestSampleImageSolid(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "solid_4285f4.png")
	colors := SampleImage(img, 0)
	require.Len(t, colors, 1)
	c := colors[0]
	require.Equal(t, uint8(0x42), c.R)
	require.Equal(t, uint8(0x85), c.G)
	require.Equal(t, uint8(0xf4), c.B)
}

func TestSampleImageMaxSamplesCapsVisits(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "gradient_256.png")
	// Full unique set is huge; striding must return fewer or equal uniques and never panic.
	limited := SampleImage(img, 256)
	full := SampleImage(img, 0)
	require.NotEmpty(t, limited, "limited sample empty")
	require.LessOrEqual(t, len(limited), len(full))
}

// bliss.jpg is the real wallpaper from ~/.dotfiles/assets/wallpapers (4510x3627).
func TestSampleImageBliss(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "bliss.jpg")
	b := img.Bounds()
	require.GreaterOrEqual(t, b.Dx(), 1000)
	require.GreaterOrEqual(t, b.Dy(), 1000)
	// CLI default-style budget: must yield some opaque colors, stay bounded.
	colors := SampleImage(img, 10000)
	require.NotEmpty(t, colors, "bliss sample empty")
	require.LessOrEqual(t, len(colors), 10000)
}

func BenchmarkSampleImage(b *testing.B) {
	b.Run("gradient_256/full", func(b *testing.B) {
		img := palettetest.LoadImage(b, "gradient_256.png")
		b.ReportAllocs()
		for b.Loop() {
			res := SampleImage(img, 0)
			if len(res) == 0 {
				// name and reference result (non-error) so blank not on call LHS
			}
		}
	})
	b.Run("gradient_256/max_10000", func(b *testing.B) {
		img := palettetest.LoadImage(b, "gradient_256.png")
		b.ReportAllocs()
		for b.Loop() {
			res := SampleImage(img, 10000)
			if len(res) == 0 {
				// name and reference result (non-error) so blank not on call LHS
			}
		}
	})
	b.Run("bliss/max_10000", func(b *testing.B) {
		img := palettetest.LoadImage(b, "bliss.jpg")
		b.ReportAllocs()
		for b.Loop() {
			res := SampleImage(img, 10000)
			if len(res) == 0 {
				// name and reference result (non-error) so blank not on call LHS
			}
		}
	})
	b.Run("bliss/max_1000", func(b *testing.B) {
		img := palettetest.LoadImage(b, "bliss.jpg")
		b.ReportAllocs()
		for b.Loop() {
			res := SampleImage(img, 1000)
			if len(res) == 0 {
				// name and reference result (non-error) so blank not on call LHS
			}
		}
	})
}
