package rofi

import (
	"context"

	"github.com/lucasew/workspaced/pkg/driver/dialog"
	execdriver "github.com/lucasew/workspaced/pkg/driver/exec"
)

func init() {
	dialog.RegisterChooserAndDriver("rofi", "Rofi", checkRofi, func() dialog.Driver { return &Driver{} })
}

func checkRofi(ctx context.Context) error {
	return dialog.RequireDisplayBinary(ctx, "rofi")
}

type Driver struct{}

func (d *Driver) Choose(ctx context.Context, opts dialog.ChooseOptions) (*dialog.Item, error) {
	return dialog.ChooseViaCmd(ctx, opts, "rofi", true, "-dmenu", "-p", opts.Prompt, "-show-icons")
}

func (d *Driver) RunApp(ctx context.Context) error {
	return execdriver.MustRun(ctx, "rofi", "-show", "combi", "-combi-modi", "drun", "-show-icons").Run()
}

func (d *Driver) SwitchWindow(ctx context.Context) error {
	return execdriver.MustRun(ctx, "rofi", "-show", "combi", "-combi-modi", "window", "-show-icons").Run()
}
