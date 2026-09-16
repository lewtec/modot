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
