package screen

import (
	"context"

	lewscreen "github.com/lewtec/lewkit/x/driver/screen"
	"github.com/lewtec/modot/internal/logging"
)

type Command struct {
	On     *On
	Off    *Off
	Toggle *Toggle
	Reset  *Reset
}

func (Command) Description() string {
	return "Screen and power management"
}

type On struct{}

func (On) Description() string { return "Turn on the screen (DPMS)" }
func (*On) Run(ctx context.Context) error {
	logging.GetLogger(ctx).Info("setting DPMS", "on", true)
	return lewscreen.SetDPMS(ctx, true)
}

type Off struct{}

func (Off) Description() string { return "Turn off the screen (DPMS)" }
func (*Off) Run(ctx context.Context) error {
	logging.GetLogger(ctx).Info("setting DPMS", "on", false)
	return lewscreen.SetDPMS(ctx, false)
}

type Toggle struct{}

func (Toggle) Description() string { return "Toggle screen state (DPMS)" }
func (*Toggle) Run(ctx context.Context) error {
	return lewscreen.ToggleDPMS(ctx)
}

type Reset struct{}

func (Reset) Description() string { return "Reset screen resolution based on host" }
func (*Reset) Run(ctx context.Context) error {
	logging.GetLogger(ctx).Info("resetting screen layout")
	return lewscreen.Reset(ctx)
}
