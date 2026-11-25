package screenshot

import (
	"strconv"
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestParseRectParts(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		r, err := ParseRectParts([]string{"10", "20", "300", "400"})
		require.NoError(t, err)
		require.Equal(t, 10, r.X)
		require.Equal(t, 20, r.Y)
		require.Equal(t, 300, r.Width)
		require.Equal(t, 400, r.Height)
	})

	t.Run("wrong count", func(t *testing.T) {
		t.Parallel()
		_, err := ParseRectParts([]string{"1", "2", "3"})
		require.Error(t, err, "expected error for wrong field count")
	})

	t.Run("non-integer", func(t *testing.T) {
		t.Parallel()
		_, err := ParseRectParts([]string{"10", "20", "abc", "400"})
		require.Error(t, err, "expected error for non-integer field")
		var ne *strconv.NumError
		require.ErrorAs(t, err, &ne)
		require.Equal(t, "abc", ne.Num)
	})
}

func TestResolveRectUnknownTarget(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	_, err := ResolveRect(ctx, TargetType(99))
	require.ErrorIs(t, err, ErrUnknownTargetType)
}
