package audio

import (
	"context"

	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/driver/notification"
	"github.com/lewtec/modot/internal/logging"
)

type Command struct {
	Up     *Up
	Down   *Down
	Mute   *Mute
	Show   *Show
	Status *Show `cmd:"status"`
}

func (Command) Description() string {
	return "Control audio volume"
}

type Up struct{}

func (Up) Description() string { return "Increase volume" }
func (*Up) Run(ctx context.Context) error {
	return adjustVolume(ctx, 0.05)
}

type Down struct{}

func (Down) Description() string { return "Decrease volume" }
func (*Down) Run(ctx context.Context) error {
	return adjustVolume(ctx, -0.05)
}

type Mute struct{}

func (Mute) Description() string { return "Toggle mute" }
func (*Mute) Run(ctx context.Context) error {
	if err := volume.ToggleMute(ctx); err != nil {
		return err
	}
	return showVolume(ctx)
}

type Show struct{}

func (Show) Description() string { return "Show current volume" }
func (*Show) Run(ctx context.Context) error {
	return showVolume(ctx)
}

func adjustVolume(ctx context.Context, delta float64) error {
	vol, err := volume.GetVolume(ctx)
	if err != nil {
		return err
	}
	if err := volume.SetVolume(ctx, driver.Clamp01(vol+delta)); err != nil {
		return err
	}
	return showVolume(ctx)
}

func showVolume(ctx context.Context) error {
	level, err := volume.GetVolume(ctx)
	if err != nil {
		return err
	}
	isMuted, err := volume.GetMute(ctx)
	if err != nil {
		return err
	}
	sinkName, err := volume.SinkName(ctx)
	if err != nil {
		return err
	}

	icon := "audio-volume-high"
	if isMuted || level == 0 {
		icon = "audio-volume-muted"
	} else if level < .33 {
		icon = "audio-volume-low"
	} else if level < .66 {
		icon = "audio-volume-medium"
	}

	logger := logging.GetLogger(ctx)
	logger.Info("volume updated", "level", level, "sink", sinkName, "muted", isMuted)

	return lewnotify.Notify(ctx, notification.Notification{
		ID:          notification.StatusNotificationID,
		Title:       "Volume",
		Message:     sinkName,
		Icon:        icon,
		Progress:    level,
		HasProgress: true,
	})
}
