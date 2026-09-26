package svc

import (
	"context"
	"fmt"
	"time"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/modot/internal/driver/battery"
	"github.com/lewtec/modot/internal/logging"
)

type Osmardetector struct{}

func (Osmardetector) Description() string {
	return "Annoying beep each second if laptop stops charging"
}

func (*Osmardetector) Run(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	logger := logging.GetLogger(ctx)
	logger.Info("osmardetector started")
	drv, err := lewdriver.Get[battery.Driver](ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := drv.BatteryStatus(ctx)
			if err != nil {
				logger.Error("failed to get battery status", "error", err)
				continue
			}
			if status == battery.Discharging {
				fmt.Print("\aAi!")
			}
		}
	}
}
