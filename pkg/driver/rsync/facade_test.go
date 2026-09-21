package rsync

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
)

func TestBindStreamsTeesExtraOut(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	var extra bytes.Buffer
	w, done := BindStreams(ctx, &extra)
	t.Cleanup(func() { done() })
	if w == os.Stderr {
		t.Fatal("writer is os.Stderr")
	}
	if _, err := io.WriteString(w, "file.txt\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	done()
	if got := extra.String(); got != "file.txt\n" {
		t.Fatalf("extra=%q want %q", got, "file.txt\n")
	}
}

func TestBindStreamsWithoutExtraIsNotProcessStderr(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	w, done := BindStreams(ctx, nil)
	t.Cleanup(done)
	if w == os.Stderr {
		t.Fatal("writer is os.Stderr")
	}
}
