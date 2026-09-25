package screen

import (
	"context"

	kitscreen "github.com/lewtec/lewkit/x/driver/screen"
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
	logger.Info("setting DPMS", "on", on)
	return kitscreen.SetDPMS(ctx, on)
}

func ToggleDPMS(ctx context.Context) error {
	return kitscreen.ToggleDPMS(ctx)
}

func IsDPMSOn(ctx context.Context) (bool, error) {
	return kitscreen.IsDPMSOn(ctx)
}

func Reset(ctx context.Context) error {
	logger := logging.GetLogger(ctx)
	logger.Info("resetting screen layout")
	return kitscreen.Reset(ctx)
}
