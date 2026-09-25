package screen

import (
	"context"
	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/driver/power"
	"github.com/lewtec/modot/internal/logging"
)

func Lock(ctx context.Context) error {
	logger := logging.GetLogger(ctx)
	logger.Info("locking session")
	return power.Lock(ctx)
}

func SetDPMS(ctx context.Context, on bool) error {
	logger := logging.GetLogger(ctx)
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return err
	}
	logger.Info("setting DPMS", "on", on)
	return d.SetDPMS(ctx, on)
}

func ToggleDPMS(ctx context.Context) error {
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return err
	}
	isOn, err := d.IsDPMSOn(ctx)
	if err != nil {
		return err
	}
	return d.SetDPMS(ctx, !isOn)
}

func IsDPMSOn(ctx context.Context) (bool, error) {
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return false, err
	}
	return d.IsDPMSOn(ctx)
}

func Reset(ctx context.Context) error {
	logger := logging.GetLogger(ctx)
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return err
	}
	logger.Info("resetting screen layout")
	return d.Reset(ctx)
}
