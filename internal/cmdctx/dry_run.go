package cmdctx

import "context"

type dryRunKey struct{}

type dryRunBox struct {
	on bool
}

func WithDryRun(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, dryRunKey{}, &dryRunBox{on: enabled})
}

// SetDryRun flips the flag stored by WithDryRun. Tasks inherit that box
// from the session ctx, so plan can force dry-run after Enter.
func SetDryRun(ctx context.Context, enabled bool) {
	if b, ok := ctx.Value(dryRunKey{}).(*dryRunBox); ok && b != nil {
		b.on = enabled
	}
}

func IsDryRun(ctx context.Context) bool {
	b, ok := ctx.Value(dryRunKey{}).(*dryRunBox)
	return ok && b != nil && b.on
}
