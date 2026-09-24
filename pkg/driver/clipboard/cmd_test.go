package clipboard

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	_ "github.com/lucasew/workspaced/pkg/driver/exec/native"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestCmdFactoryCompat(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "clipfake")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx := logging.NewWriterContext(t.Output())

	missing := &cmdFactory{binary: "definitely-missing-clip"}
	err := missing.CheckCompatibility(ctx)
	require.ErrorIs(t, err, driver.ErrIncompatible)

	okFactory := &cmdFactory{binary: "clipfake"}
	require.NoError(t, okFactory.CheckCompatibility(ctx))

	envFirst := &cmdFactory{
		binary: "clipfake",
		compat: func(ctx context.Context) error {
			return driver.RequireEnv(ctx, "WAYLAND_DISPLAY")
		},
	}
	t.Setenv("WAYLAND_DISPLAY", "")
	err = envFirst.CheckCompatibility(ctx)
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.Contains(t, err.Error(), "WAYLAND_DISPLAY")

	t.Setenv("WAYLAND_DISPLAY", "1")
	require.NoError(t, envFirst.CheckCompatibility(ctx))
}

func TestCmdDriverWritesPayload(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "clipfake")
	body := "#!/bin/sh\nprintf '%s\\0' \"$@\" > \"$CLIP_ARGS\"\ncat > \"$CLIP_STDIN\"\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx := logging.NewWriterContext(t.Output())

	d := &cmdDriver{
		binary:    "clipfake",
		imageArgs: []string{"-t", "image/png"},
		textArgs:  []string{"-selection", "clipboard"},
	}

	t.Run("text", func(t *testing.T) {
		argsPath := filepath.Join(dir, "text.args")
		stdinPath := filepath.Join(dir, "text.stdin")
		t.Setenv("CLIP_ARGS", argsPath)
		t.Setenv("CLIP_STDIN", stdinPath)
		require.NoError(t, d.WriteText(ctx, "hello"))
		require.Equal(t, []string{"-selection", "clipboard"}, splitNull(mustRead(t, argsPath)))
		require.Equal(t, "hello", string(mustRead(t, stdinPath)))
	})

	t.Run("image", func(t *testing.T) {
		argsPath := filepath.Join(dir, "image.args")
		stdinPath := filepath.Join(dir, "image.stdin")
		t.Setenv("CLIP_ARGS", argsPath)
		t.Setenv("CLIP_STDIN", stdinPath)
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		require.NoError(t, d.WriteImage(ctx, img))
		require.Equal(t, []string{"-t", "image/png"}, splitNull(mustRead(t, argsPath)))
		got, err := png.Decode(bytes.NewReader(mustRead(t, stdinPath)))
		require.NoError(t, err)
		c := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
		require.Equal(t, color.RGBA{R: 10, G: 20, B: 30, A: 255}, c)
	})
}

func splitNull(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	parts := strings.Split(string(b), "\x00")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return b
}
