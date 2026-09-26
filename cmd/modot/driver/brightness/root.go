package brightness

import (
	"context"

	lewbrightness "github.com/lewtec/lewkit/x/driver/brightness"
	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
)

type Command struct {
	Up     *Up
	Down   *Down
	Show   *Show
	Status *Show `cmd:"status"`
}

func (Command) Description() string {
	return "Control screen brightness"
}

type Up struct{}

func (Up) Description() string { return "Increase brightness" }
func (*Up) Run(ctx context.Context) error {
	if err := lewbrightness.Increase(ctx); err != nil {
		return err
	}
	return showBrightness(ctx)
}

type Down struct{}

func (Down) Description() string { return "Decrease brightness" }
func (*Down) Run(ctx context.Context) error {
	if err := lewbrightness.Decrease(ctx); err != nil {
		return err
	}
	return showBrightness(ctx)
}

type Show struct{}

func (Show) Description() string { return "Show current brightness" }
func (*Show) Run(ctx context.Context) error {
	return showBrightness(ctx)
}

func showBrightness(ctx context.Context) error {
	status, err := lewbrightness.Status(ctx)
	if err != nil {
		return err
	}
	return lewnotify.Notify(ctx, lewbrightness.StatusNotification(status.Name, status.Brightness))
}
