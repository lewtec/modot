package wlcopy

import (
	"context"

	"github.com/lucasew/workspaced/pkg/driver"
	"github.com/lucasew/workspaced/pkg/driver/clipboard"
)

func init() {
	clipboard.RegisterCmd(
		"clipboard_wlcopy",
		"Wayland (wl-copy)",
		"wl-copy",
		func(ctx context.Context) error {
			return driver.RequireEnv(ctx, "WAYLAND_DISPLAY")
		},
		[]string{"-t", "image/png"},
		nil,
	)
}
