package termux

import (
	"context"
	"fmt"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	kitpower "github.com/lewtec/lewkit/x/driver/power"
	"github.com/lewtec/modot/internal/api"
	"github.com/lewtec/modot/internal/driver"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

func init() {
	lewdriver.Register[kitpower.Driver](factory{})
}

type factory struct{}

func (factory) ID() string   { return "power_termux" }
func (factory) Name() string { return "Termux" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.RequireTermux()
}

func (factory) New(context.Context) (kitpower.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Lock(context.Context) error {
	return fmt.Errorf("%w: screen lock not possible in Termux", api.ErrNotSupported)
}

func (backend) Logout(context.Context) error {
	return fmt.Errorf("%w: logout not possible in Termux", api.ErrNotSupported)
}

func (backend) Suspend(context.Context) error {
	return fmt.Errorf("%w: suspend not possible in Termux", api.ErrNotSupported)
}

func (backend) Hibernate(context.Context) error {
	return fmt.Errorf("%w: hibernate not possible in Termux", api.ErrNotSupported)
}

func (backend) Reboot(ctx context.Context) error {
	return execdriver.MustRun(ctx, "reboot").Run()
}

func (backend) Shutdown(ctx context.Context) error {
	return execdriver.MustRun(ctx, "shutdown", "-h", "now").Run()
}
