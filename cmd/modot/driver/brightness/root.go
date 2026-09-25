package brightness

import (
	"context"

	lewbrightness "github.com/lewtec/lewkit/x/driver/brightness"
	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/driver/notification"
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
	return adjustBrightness(ctx, 0.05)
}

type Down struct{}

func (Down) Description() string { return "Decrease brightness" }
func (*Down) Run(ctx context.Context) error {
	return adjustBrightness(ctx, -0.05)
}

type Show struct{}

func (Show) Description() string { return "Show current brightness" }
func (*Show) Run(ctx context.Context) error {
	return showBrightness(ctx)
}

func adjustBrightness(ctx context.Context, delta float64) error {
	status, err := lewbrightness.Status(ctx)
	if err != nil {
		return err
	}
	if err := lewbrightness.SetBrightness(ctx, driver.Clamp01(status.Brightness+delta)); err != nil {
		return err
	}
	return showBrightness(ctx)
}

func showBrightness(ctx context.Context) error {
	status, err := lewbrightness.Status(ctx)
	if err != nil {
		return err
	}
	return lewnotify.Notify(ctx, notification.Notification{
		ID:          notification.StatusNotificationID,
		Title:       "Brightness",
		Message:     status.Name,
		Icon:        "display-brightness",
		Progress:    status.Brightness,
		HasProgress: true,
	})
}
