package resolution

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestReadToolVersion_found(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	path := filepath.Join(dir, ".tool-versions")
	require.NoError(t, os.WriteFile(path, []byte("# comment\n\ngo 1.22.0\ndeno 1.40.0\n"), 0o644))

	got, err := readToolVersion(ctx, path, "deno")
	require.NoError(t, err)
	require.Equal(t, "1.40.0", got)
}

func TestReadToolVersion_missing(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	path := filepath.Join(dir, ".tool-versions")
	require.NoError(t, os.WriteFile(path, []byte("go 1.22.0\n"), 0o644))

	got, err := readToolVersion(ctx, path, "deno")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestReadToolVersion_tokenTooLong(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	dir := t.TempDir()
	path := filepath.Join(dir, ".tool-versions")
	// bufio.Scanner default max token size is 64KiB; one line longer than that fails.
	longLine := strings.Repeat("x", 70*1024) + "\n"
	require.NoError(t, os.WriteFile(path, []byte(longLine), 0o644))

	got, err := readToolVersion(ctx, path, "deno")
	require.Error(t, err, "expected scan error for oversized line")
	require.Empty(t, got)
	require.ErrorIs(t, err, bufio.ErrTooLong)
}

func TestReadToolVersion_openMissing(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	_, err := readToolVersion(ctx, filepath.Join(t.TempDir(), "nope"), "deno")
	require.Error(t, err, "expected open error")
}
