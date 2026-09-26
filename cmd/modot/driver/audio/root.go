package audio

import (
	"context"

	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/lewkit/x/driver/volume"
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
	if err := volume.Increase(ctx); err != nil {
		return err
	}
	return showVolume(ctx)
}

type Down struct{}

func (Down) Description() string { return "Decrease volume" }
func (*Down) Run(ctx context.Context) error {
	if err := volume.Decrease(ctx); err != nil {
		return err
	}
	return showVolume(ctx)
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

func showVolume(ctx context.Context) error {
	level, err := volume.GetVolume(ctx)
	if err != nil {
		return err
	}
	muted, err := volume.GetMute(ctx)
	if err != nil {
		return err
	}
	sink, err := volume.SinkName(ctx)
	if err != nil {
		return err
	}
	logging.GetLogger(ctx).Info("volume updated", "level", level, "sink", sink, "muted", muted)
	return lewnotify.Notify(ctx, volume.StatusNotification(level, muted, sink))
}
