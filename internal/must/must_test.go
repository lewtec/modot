package must_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lucasew/workspaced/internal/must"
)

func TestMustOK(t *testing.T) {
	t.Parallel()
	must.Must(func() error { return nil })
}

func TestMustPanics(t *testing.T) {
	t.Parallel()
	errBoom := errors.New("boom")
	defer func() {
		r := recover()
		require.Equal(t, errBoom, r)
	}()
	must.Must(func() error { return errBoom })
	require.Fail(t, "expected panic")
}

func TestMustContext(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	var saw context.Context
	must.MustContext(ctx, func(c context.Context) error {
		saw = c
		return nil
	})
	require.Equal(t, ctx, saw, "fn did not receive ctx")

	errBoom := errors.New("boom")
	defer func() {
		require.Equal(t, errBoom, recover())
	}()
	must.MustContext(ctx, func(context.Context) error { return errBoom })
	require.Fail(t, "expected panic")
}

func TestValueAndValueContext(t *testing.T) {
	t.Parallel()
	require.Equal(t, 7, must.Value(func() (int, error) { return 7, nil }))
	ctx := t.Context()
	require.Equal(t, "ok", must.ValueContext(ctx, func(context.Context) (string, error) { return "ok", nil }))
}

func TestErr(t *testing.T) {
	t.Parallel()
	must.Err(nil)
	errBoom := errors.New("boom")
	defer func() {
		require.Equal(t, errBoom, recover())
	}()
	must.Err(errBoom)
	require.Fail(t, "expected panic")
}
