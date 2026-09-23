package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeUnknownCodec(t *testing.T) {
	_, err := Decode("not_a_codec", "tool", nil)
	require.Error(t, err, "expected error for unknown codec")
	require.ErrorIs(t, err, ErrUnknownCodec)
}
