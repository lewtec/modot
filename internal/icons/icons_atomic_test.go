package icons

import (
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritePNGFileAtomic_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "icon-cache.png")

	img := solidNRGBA(2, 2, color.NRGBA{R: 0xff, A: 0xff})
	require.NoError(t, writePNGFileAtomic(path, img))

	got, err := decodePNG(path)
	require.NoError(t, err)
	require.Equal(t, 2, got.Bounds().Dx(), "decoded size = %v, want 2x2", got.Bounds())
	require.Equal(t, 2, got.Bounds().Dy(), "decoded size = %v, want 2x2", got.Bounds())
}

func TestWritePNGFileAtomic_FailureKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "icon-cache.png")

	prior := solidNRGBA(1, 1, color.NRGBA{G: 0xff, A: 0xff})
	require.NoError(t, writePNGFileAtomic(path, prior))
	priorBytes, err := os.ReadFile(path)
	require.NoError(t, err)

	// Make dir non-writable so Create cannot open a new temp.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() {
		assert.NoError(t, os.Chmod(dir, 0o755), "chmod restore")
	})

	err = writePNGFileAtomic(path, solidNRGBA(3, 3, color.NRGBA{B: 0xff, A: 0xff}))
	require.Error(t, err, "expected error when parent dir is not writable")

	require.NoError(t, os.Chmod(dir, 0o755))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, priorBytes, got, "existing cache was mutated")
}

func TestWritePNGFileAtomic_FailureLeavesNoFinal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "icon-cache.png")

	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() {
		assert.NoError(t, os.Chmod(dir, 0o755), "chmod restore")
	})

	err := writePNGFileAtomic(path, solidNRGBA(2, 2, color.NRGBA{R: 1, A: 0xff}))
	require.Error(t, err, "expected error when parent dir is not writable")
	require.NoError(t, os.Chmod(dir, 0o755))
	_, err = os.Stat(path)
	require.ErrorIs(t, err, fs.ErrNotExist, "final path should not exist after failed first write")
}

func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func decodePNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
