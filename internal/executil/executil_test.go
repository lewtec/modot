package executil

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/pkg/logging"
)

func TestInheritContextWritersDoesNotUseProcessStreams(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	cmd := exec.Command("true")
	InheritContextWriters(ctx, cmd)
	if cmd.Stderr == os.Stderr {
		t.Fatal("stderr is os.Stderr")
	}
	if cmd.Stdout == os.Stdout || cmd.Stdout == os.Stderr {
		t.Fatal("stdout is a process stream")
	}
	if cmd.Stdout != cmd.Stderr {
		t.Fatal("stdout should share the live-row writer with stderr")
	}
}

func TestInheritContextWritersKeepsExistingStderr(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	cmd := exec.Command("true")
	live := taskgroup.LineWriterFrom(ctx)
	t.Cleanup(func() {
		if err := live.Close(); err != nil {
			t.Errorf("close live writer: %v", err)
		}
	})
	cmd.Stderr = live
	InheritContextWriters(ctx, cmd)
	if cmd.Stderr != live {
		t.Fatal("replaced existing stderr writer")
	}
	if cmd.Stdout != live {
		t.Fatal("stdout should reuse existing stderr writer")
	}
}

func TestInheritContextWritersHonorsContextWriters(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	ctx := logging.NewWriterContext(t.Output())
	ctx = WithStdout(ctx, &stdout)
	ctx = WithStderr(ctx, &stderr)
	cmd := exec.Command("true")
	InheritContextWriters(ctx, cmd)
	if cmd.Stdout != &stdout {
		t.Fatal("stdout is not the context writer")
	}
	if cmd.Stderr != &stderr {
		t.Fatal("stderr is not the context writer")
	}
}
