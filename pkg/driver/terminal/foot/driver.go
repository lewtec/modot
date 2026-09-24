package foot

import (
	"context"

	"github.com/lucasew/workspaced/pkg/driver"
	"github.com/lucasew/workspaced/pkg/driver/terminal"
)

func init() {
	terminal.RegisterExec("terminal_foot", "Foot", "foot", "-T", false, func(ctx context.Context) error {
		return driver.RequireEnv(ctx, "WAYLAND_DISPLAY")
	})
}
