package native

import (
	"log/slog"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestExecRsyncBinaryNotAvailable(t *testing.T) {
	t.Parallel()
	// No exec driver on ctx → IsBinaryAvailable is false.
	ctx := logging.NewWriterContext(t.Output())
	err := (&Driver{}).execRsync(ctx, []string{"-av", "a/", "b/"}, nil, nil, slog.Default())
	require.ErrorIs(t, err, ErrBinaryNotAvailable)
}
