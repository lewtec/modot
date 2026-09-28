package screenshot

import (
	"testing"

	lewscreenshot "github.com/lewtec/lewkit/x/driver/screenshot"
	"github.com/stretchr/testify/require"
)

func TestResolveRectUnknownTarget(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, err := lewscreenshot.ResolveRect(ctx, TargetType(99))
	require.ErrorIs(t, err, ErrUnknownTargetType)
}
