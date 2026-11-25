package xdg

import (
	"context"

	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/driver/opener"
)

func init() {
	opener.RegisterBinary("opener_xdg", "xdg-open", "xdg-open", func(ctx context.Context) error {
		return driver.RequireAnyEnv(ctx, "DISPLAY", "WAYLAND_DISPLAY")
	})
}
