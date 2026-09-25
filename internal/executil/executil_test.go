package executil

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInheritContextWritersDoesNotUseProcessStreams(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	cmd := exec.Command("true")
	InheritContextWriters(ctx, cmd)
	require.False(t, cmd.Stderr == os.Stderr, "stderr is os.Stderr")
	require.False(t, cmd.Stdout == os.Stdout || cmd.Stdout == os.Stderr, "stdout is a process stream")
	require.True(t, cmd.Stdout == cmd.Stderr, "stdout should share the live-row writer with stderr")
}

func TestInheritContextWritersKeepsExistingStderr(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	cmd := exec.Command("true")
	live := taskgroup.LineWriterFrom(ctx)
	t.Cleanup(func() {
		assert.NoError(t, live.Close(), "close live writer")
	})
	cmd.Stderr = live
	InheritContextWriters(ctx, cmd)
	require.True(t, cmd.Stderr == live, "replaced existing stderr writer")
	require.True(t, cmd.Stdout == live, "stdout should reuse existing stderr writer")
}

func TestInheritContextWritersHonorsContextWriters(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	ctx := logging.NewWriterContext(t.Output())
	ctx = WithStdout(ctx, &stdout)
	ctx = WithStderr(ctx, &stderr)
	cmd := exec.Command("true")
	InheritContextWriters(ctx, cmd)
	require.True(t, cmd.Stdout == &stdout, "stdout is not the context writer")
	require.True(t, cmd.Stderr == &stderr, "stderr is not the context writer")
}
