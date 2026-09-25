package feh

import (
	"context"
	"fmt"
	"github.com/lewtec/modot/internal/driver"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	"github.com/lewtec/modot/internal/driver/wallpaper"
)

func init() {
	driver.Register[wallpaper.Driver](&Factory{})
}

type Factory struct{}

func (f *Factory) ID() string   { return "x11_feh" }
func (f *Factory) Name() string { return "X11 (feh)" }

func (f *Factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireEnvBinaries(ctx, "DISPLAY", "systemd-run", "feh")
}

func (f *Factory) New(ctx context.Context) (wallpaper.Driver, error) {
	return &Driver{}, nil
}

type Driver struct{}

func (d *Driver) SetStatic(ctx context.Context, path string) error {
	feh, err := execdriver.Which(ctx, "feh")
	if err != nil {
		return err
	}
	cmd := execdriver.MustRun(ctx, "systemd-run", "--user", "-u", "wallpaper-change", "--collect", feh, "--bg-fill", path)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("can't run feh in systemd unit: %w", err)
	}
	return nil
}
