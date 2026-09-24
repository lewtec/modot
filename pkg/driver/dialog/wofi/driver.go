package wofi

import (
	"context"

	"github.com/lucasew/workspaced/pkg/driver/dialog"
	execdriver "github.com/lucasew/workspaced/pkg/driver/exec"
)

func init() {
	dialog.RegisterChooserAndDriver("wofi", "Wofi", checkWofi, func() dialog.Driver { return &Driver{} })
}

func checkWofi(ctx context.Context) error {
	return execdriver.RequireEnvBinary(ctx, "WAYLAND_DISPLAY", "wofi")
}

type Driver struct{}

func (d *Driver) Choose(ctx context.Context, opts dialog.ChooseOptions) (*dialog.Item, error) {
	return dialog.ChooseViaCmd(ctx, opts, "wofi", false, "--dmenu", "-p", opts.Prompt)
}

func (d *Driver) RunApp(ctx context.Context) error {
	// wofi has no dedicated window switcher mode; reuse the app launcher.
	return execdriver.MustRun(ctx, "wofi", "--show", "drun").Run()
}

func (d *Driver) SwitchWindow(ctx context.Context) error {
	return d.RunApp(ctx)
}
