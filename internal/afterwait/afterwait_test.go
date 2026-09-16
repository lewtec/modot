package afterwait

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterAndRun(t *testing.T) {
	t.Parallel()
	ctx := With(t.Context())
	var n atomic.Int32
	Register(ctx, func() error {
		n.Add(1)
		return nil
	})
	require.NoError(t, Run(ctx))
	require.Equal(t, int32(1), n.Load())
}

func TestRegisterWithoutWithIsNoop(t *testing.T) {
	t.Parallel()
	Register(t.Context(), func() error {
		t.Fatal("hook ran without With")
		return nil
	})
	require.NoError(t, Run(t.Context()))
}

func TestRunReturnsFirstError(t *testing.T) {
	t.Parallel()
	ctx := With(t.Context())
	Register(ctx, func() error { return errHook })
	Register(ctx, func() error { return nil })
	require.ErrorIs(t, Run(ctx), errHook)
}

var errHook = errors.New("hook fail")
