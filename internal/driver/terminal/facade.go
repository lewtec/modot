package terminal

import (
	"context"

	kitterminal "github.com/lewtec/lewkit/x/driver/terminal"
)

func Open(ctx context.Context, opts Options) error {
	return kitterminal.Open(ctx, opts)
}
