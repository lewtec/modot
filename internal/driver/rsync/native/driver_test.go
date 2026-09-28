package native

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecRsyncBinaryNotAvailable(t *testing.T) {
	t.Parallel()
	// No exec driver on ctx → IsBinaryAvailable is false.
	ctx := t.Context()
	err := (&Driver{}).execRsync(ctx, []string{"-av", "a/", "b/"}, nil, nil)
	require.ErrorIs(t, err, ErrBinaryNotAvailable)
}
