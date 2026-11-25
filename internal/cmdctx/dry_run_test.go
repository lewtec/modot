package cmdctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetDryRunMutatesInheritedBox(t *testing.T) {
	t.Parallel()
	ctx := WithDryRun(t.Context(), false)
	type k struct{}
	child := context.WithValue(ctx, k{}, 1)
	require.False(t, IsDryRun(child))
	SetDryRun(ctx, true)
	require.True(t, IsDryRun(child))
}
