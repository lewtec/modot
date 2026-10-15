package termux

import (
	"context"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/modot/internal/driver/opener"
)

func init() {
	opener.RegisterBinary("opener_termux", "termux-open", "termux-open", func(context.Context) error {
		return lewdriver.RequireTermux()
	})
}
