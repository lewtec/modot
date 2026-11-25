package cmdctx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithNoCache(t *testing.T) {
	ctx := t.Context()
	require.False(t, IsNoCache(ctx), "default off")
	ctx = WithNoCache(ctx, true)
	require.True(t, IsNoCache(ctx), "expected on")
	ctx = WithNoCache(ctx, false)
	require.False(t, IsNoCache(ctx), "expected off")
}
