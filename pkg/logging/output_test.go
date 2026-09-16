package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetProcessWriter(t *testing.T) {
	t.Cleanup(func() { SetProcessWriter(nil) })
	var buf bytes.Buffer
	SetProcessWriter(&buf)
	_, err := ProcessWriter().Write([]byte("hi"))
	require.NoError(t, err)
	require.Equal(t, "hi", buf.String())
}
