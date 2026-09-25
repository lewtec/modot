package screenshot

import (
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestResolveRectUnknownTarget(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	_, err := ResolveRect(ctx, TargetType(99))
	require.ErrorIs(t, err, ErrUnknownTargetType)
}
