package svc

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	lewscreen "github.com/lewtec/lewkit/x/driver/screen"
)

type Screencaps struct{}

func (Screencaps) Description() string {
	return "Monitor CapsLock and toggle screen DPMS"
}

func (*Screencaps) Run(ctx context.Context) error {
	monitorCapsLock(ctx)
	return nil
}

func monitorCapsLock(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	matches, err := filepath.Glob("/sys/class/leds/*capslock/brightness")
	if err != nil || len(matches) == 0 {
		logger := slog.Default()
		logger.Warn("no capslock leds found")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			capsActive := false
			for _, m := range matches {
				data, err := os.ReadFile(m)
				if err == nil && strings.TrimSpace(string(data)) == "1" {
					capsActive = true
					break
				}
			}

			logger := slog.Default()
			screenActive, err := lewscreen.IsDPMSOn(ctx)
			if err != nil {
				logger.Error("on checking if screen is active", "error", err)
			}
			if !capsActive != screenActive {
				logger.Info("toggling screen", "active", !capsActive)
				if err := lewscreen.SetDPMS(ctx, !capsActive); err != nil {
					logger.Error("failed to set screen DPMS", "active", !capsActive, "error", err)
				}
			}
		}
	}
}
