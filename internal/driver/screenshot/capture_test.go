package screenshot

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestCaptureViaCmd(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())

	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	path := filepath.Join(t.TempDir(), "shot.png")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))

	got, err := CaptureViaCmd(ctx, "cat", path)
	require.NoError(t, err)
	require.Equal(t, img.Bounds(), got.Bounds())
	c := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	require.Equal(t, uint8(255), c.R, "pixel(0,0)=%v want red", c)
	require.Equal(t, uint8(255), c.A, "pixel(0,0)=%v want red", c)
}

func TestCaptureViaCmdFailed(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	_, err := CaptureViaCmd(ctx, "false")
	require.Error(t, err, "expected command failure")
	require.ErrorAs(t, err, new(*exec.ExitError))
}

func TestCaptureViaCmdBadImage(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	path := filepath.Join(t.TempDir(), "not.png")
	require.NoError(t, os.WriteFile(path, []byte("not-an-image"), 0o644))
	_, err := CaptureViaCmd(ctx, "cat", path)
	require.Error(t, err, "expected decode failure")
	require.ErrorIs(t, err, image.ErrFormat)
}
