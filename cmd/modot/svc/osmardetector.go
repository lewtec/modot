package svc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/modot/internal/driver/battery"
)

type Osmardetector struct{}

func (Osmardetector) Description() string {
	return "Annoying beep each second if laptop stops charging"
}

func (*Osmardetector) Run(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	slog.Info("osmardetector started")
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
				slog.Error("failed to get battery status", "error", err)
				continue
			}
			if status == battery.Discharging {
				fmt.Print("\aAi!")
			}
		}
	}
}
