package audio

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/modot/internal/driver/notification"
	"github.com/lewtec/modot/internal/logging"
)

// ShowStatus displays the default sink volume and mute state.
func ShowStatus(ctx context.Context) error {
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

	n := notification.Notification{
		ID:          notification.StatusNotificationID,
		Title:       "Volume",
		Message:     sinkName,
		Icon:        icon,
		Progress:    level,
		HasProgress: true,
	}
	return notification.Notify(ctx, &n)
}
