package swaybg

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

func (f *Factory) ID() string   { return "wayland_swaybg" }
func (f *Factory) Name() string { return "Wayland (swaybg)" }

func (f *Factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireEnvBinaries(ctx, "WAYLAND_DISPLAY", "systemd-run", "swaybg")
}

func (f *Factory) New(ctx context.Context) (wallpaper.Driver, error) {
	return &Driver{}, nil
}

type Driver struct{}

func (d *Driver) SetStatic(ctx context.Context, path string) error {
	swaybg, err := execdriver.Which(ctx, "swaybg")
	if err != nil {
		return err
	}

	if err = execdriver.MustRun(ctx, "systemd-run", "--user", "-u", "wallpaper-change", "--collect", swaybg, "-i", path).Run(); err != nil {
		return fmt.Errorf("can't run swaybg in systemd unit: %w", err)
	}
	return nil
}
