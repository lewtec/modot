package genetic

import (
	"math/rand"
	"os"
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/palette/api"
	"github.com/lewtec/modot/internal/palette/palettetest"
	"github.com/stretchr/testify/require"
)

func TestGeneticScoreFromBlocksTestdata(t *testing.T) {
	t.Parallel()
	// Cheap match path: sample fixture colors and score a tiny population.
	// Full Extract is opt-in (MODOT_TEST_GENETIC_EXTRACT=1) / benchmarks only.
	img := palettetest.LoadImage(t, "blocks_64.png")
	colors := api.SampleImage(img, 0)
	require.Len(t, colors, 4)
	lab := make([]api.LAB, len(colors))
	for i, c := range colors {
		lab[i] = api.RGBToLAB(c)
	}
	pop := initPopulation(rand.New(rand.NewSource(42)), 16, 32)
	scored := scorePop(pop, lab, api.PolarityDark)
	require.Len(t, scored, 32)
	require.GreaterOrEqual(t, scored[0].fitness, scored[len(scored)-1].fitness, "expected scorePop sorted by fitness descending")
	pal := mapToPalette(scored[0].individual, 16)
	require.NotEmpty(t, pal.Base00)
	require.NotEmpty(t, pal.Base0F)
}

func TestGeneticScoreFromBlissTestdata(t *testing.T) {
	t.Parallel()
	img := palettetest.LoadImage(t, "bliss.jpg")
	colors := api.SampleImage(img, 10000)
	require.NotEmpty(t, colors, "bliss sample empty")
	lab := make([]api.LAB, len(colors))
	for i, c := range colors {
		lab[i] = api.RGBToLAB(c)
	}
	pop := initPopulation(rand.New(rand.NewSource(42)), 16, 32)
	scored := scorePop(pop, lab, api.PolarityDark)
	require.GreaterOrEqual(t, scored[0].fitness, scored[len(scored)-1].fitness, "expected scorePop sorted by fitness descending")
	pal := mapToPalette(scored[0].individual, 16)
	require.NotEmpty(t, pal.Base00)
	require.NotEmpty(t, pal.Base0F)
}

func TestGeneticExtractFromTestdata(t *testing.T) {
	t.Parallel()
	if testing.Short() || os.Getenv("MODOT_TEST_GENETIC_EXTRACT") == "" {
		t.Skip("set MODOT_TEST_GENETIC_EXTRACT=1 (and not -short) for full evolution")
	}
	// Prefer real wallpaper when available; MaxSamples matches CLI default.
	img := palettetest.LoadImage(t, "bliss.jpg")
	ctx := logging.NewWriterContext(t.Output())
	d := &Driver{}
	pal, err := d.Extract(ctx, img, api.Options{
		Polarity:   api.PolarityDark,
		ColorCount: 16,
		MaxSamples: 10000,
	})
	require.NoError(t, err)
	require.NotEmpty(t, pal.Base00)
	require.NotEmpty(t, pal.Base0F)
	require.Len(t, pal.Base00, 6)
	require.NotEqual(t, byte('#'), pal.Base00[0])
}

func BenchmarkGeneticExtract(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping full genetic extract benchmark in short mode")
	}
	ctx := logging.NewWriterContext(b.Output())
	d := &Driver{}

	b.Run("blocks_64", func(b *testing.B) {
		img := palettetest.LoadImage(b, "blocks_64.png")
		opts := api.Options{Polarity: api.PolarityDark, ColorCount: 16, MaxSamples: 0}
		b.ReportAllocs()
		for b.Loop() {
			_, err := d.Extract(ctx, img, opts)
			require.NoError(b, err)
		}
	})
	b.Run("bliss", func(b *testing.B) {
		img := palettetest.LoadImage(b, "bliss.jpg")
		opts := api.Options{Polarity: api.PolarityDark, ColorCount: 16, MaxSamples: 10000}
		b.ReportAllocs()
		for b.Loop() {
			_, err := d.Extract(ctx, img, opts)
			require.NoError(b, err)
		}
	})
}

func BenchmarkGeneticScorePopFromTestdata(b *testing.B) {
	b.Run("blocks_64", func(b *testing.B) {
		img := palettetest.LoadImage(b, "blocks_64.png")
		colors := api.SampleImage(img, 0)
		lab := make([]api.LAB, len(colors))
		for i, c := range colors {
			lab[i] = api.RGBToLAB(c)
		}
		pop := initPopulation(rand.New(rand.NewSource(1)), 16, 200)
		b.ReportAllocs()
		for b.Loop() {
			res := scorePop(pop, lab, api.PolarityDark)
			var _ = res
			// named result (non-error return); var _ = to use without blank on call LHS
		}
	})
	b.Run("bliss_max_10000", func(b *testing.B) {
		img := palettetest.LoadImage(b, "bliss.jpg")
		colors := api.SampleImage(img, 10000)
		lab := make([]api.LAB, len(colors))
		for i, c := range colors {
			lab[i] = api.RGBToLAB(c)
		}
		pop := initPopulation(rand.New(rand.NewSource(1)), 16, 200)
		b.ReportAllocs()
		for b.Loop() {
			res := scorePop(pop, lab, api.PolarityDark)
			var _ = res
			// named result (non-error return); var _ = to use without blank on call LHS
		}
	})
}
