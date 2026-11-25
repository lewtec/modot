package camera

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWritePNGAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	require.NoError(t, os.WriteFile(path, []byte("not-a-png"), 0o644))
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	require.NoError(t, writePNGAtomic(path, img))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 8)
	require.Equal(t, "\x89PNG\r\n\x1a\n", string(raw[:8]))
}
