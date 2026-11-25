package cmdctx

import "context"

type noCacheKey struct{}

// WithNoCache records whether full-cascade cold materialization is armed.
func WithNoCache(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, noCacheKey{}, enabled)
}

// IsNoCache reports whether --no-cache is armed on ctx.
func IsNoCache(ctx context.Context) bool {
	v, ok := ctx.Value(noCacheKey{}).(bool)
	return ok && v
}
