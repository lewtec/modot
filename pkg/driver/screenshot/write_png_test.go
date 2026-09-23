package screenshot

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
	err := os.WriteFile(path, []byte("truncated"), 0o644)
	require.NoError(t, err)
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{G: 255, A: 255})
	require.NoError(t, writePNGAtomic(path, img))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 8)
	require.Equal(t, "\x89PNG\r\n\x1a\n", string(raw[:8]))
}
