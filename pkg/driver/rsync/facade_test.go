package rsync

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/lucasew/workspaced/internal/executil"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestBindStreamsTeesExtraOut(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	var extra bytes.Buffer
	w, done := BindStreams(ctx, &extra)
	t.Cleanup(func() { done() })
	require.False(t, w == os.Stderr, "writer is os.Stderr")
	_, err := io.WriteString(w, "file.txt\n")
	require.NoError(t, err)
	done()
	require.Equal(t, "file.txt\n", extra.String())
}

func TestBindStreamsWithoutExtraIsNotProcessStderr(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	w, done := BindStreams(ctx, nil)
	t.Cleanup(done)
	require.False(t, w == os.Stderr, "writer is os.Stderr")
}

func TestBindStreamsHonorsContextStderr(t *testing.T) {
	t.Parallel()
	var got bytes.Buffer
	ctx := executil.WithStderr(logging.NewWriterContext(t.Output()), &got)
	var extra bytes.Buffer
	w, done := BindStreams(ctx, &extra)
	t.Cleanup(done)
	_, err := io.WriteString(w, "hello\n")
	require.NoError(t, err)
	done()
	require.Equal(t, "hello\n", got.String())
	require.Equal(t, "hello\n", extra.String())
}
