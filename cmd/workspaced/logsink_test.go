package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwapWriterSet(t *testing.T) {
	var a, b bytes.Buffer
	w := newSwapWriter(&a)
	_, err := w.Write([]byte("one"))
	require.NoError(t, err)
	w.Set(&b)
	_, err = w.Write([]byte("two"))
	require.NoError(t, err)
	require.Equal(t, "one", a.String())
	require.Equal(t, "two", b.String())
}

func TestSwapWriterStaysAfterWorkReturns(t *testing.T) {
	var scheduled, duringWait bytes.Buffer
	w := newSwapWriter(&scheduled)
	_, err := w.Write([]byte("schedule\n"))
	require.NoError(t, err)
	// progress.Run Wait happens after work() returns; sink must not flip back.
	_, err = w.Write([]byte("task\n"))
	require.NoError(t, err)
	require.Equal(t, "schedule\ntask\n", scheduled.String())
	require.Empty(t, duringWait.String())
}
