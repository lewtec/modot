package media

import (
	"context"
	"fmt"
	"time"

	lewmedia "github.com/lewtec/lewkit/x/driver/media"
	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/modot/internal/driver/notification"
	"github.com/lewtec/modot/internal/logging"
)

func RunAction(ctx context.Context, action string) error {
	var err error
	switch action {
	case "next":
		err = lewmedia.Next(ctx)
	case "previous":
		err = lewmedia.Previous(ctx)
	case "play-pause":
		err = lewmedia.PlayPause(ctx)
	case "stop":
		err = lewmedia.Stop(ctx)
	case "show":
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	if err != nil {
		return err
	}
	if action == "next" || action == "previous" || action == "play-pause" {
		time.Sleep(200 * time.Millisecond)
	}
	return ShowStatus(ctx)
}

func ShowStatus(ctx context.Context) error {
	meta, err := lewmedia.GetMetadata(ctx)
	if err != nil {
		return err
	}
	return Notify(ctx, meta)
}

func Notify(ctx context.Context, meta *Metadata) error {
	if meta == nil || meta.Title == "" {
		logger := logging.GetLogger(ctx)
		logger.Warn("no active player with title found")
		return nil
	}

	progress := 0.0
	if meta.Length > 0 {
		progress = float64(meta.Position) / float64(meta.Length)
	}

	iconPath := ""
	if meta.ArtUrl != "" {
		var err error
		iconPath, err = GetArtCachePath(ctx, meta.ArtUrl)
		if err != nil {
			logging.ReportError(ctx, err)
		}
	}

	title := meta.Title
	if title == "" {
		title = "Unknown Track"
	}
	message := meta.Artist
	if message == "" {
		message = "Unknown Artist"
	}

	n := notification.Notification{
		ID:          notification.StatusNotificationID,
		Title:       title,
		Message:     message,
		Icon:        iconPath,
		Progress:    progress,
		HasProgress: true,
	}

	logger := logging.GetLogger(ctx)
	logger.Info("sending media notification",
		"player", meta.Player,
		"title", title,
		"artist", message,
		"progress", progress,
		"icon", iconPath,
	)

	return lewnotify.Notify(ctx, n)
}

func Watch(ctx context.Context) {
	err := lewmedia.Watch(ctx, func(meta *Metadata) {
		if err := Notify(ctx, meta); err != nil {
			logging.ReportError(ctx, err)
		}
	})
	if err != nil {
		logger := logging.GetLogger(ctx)
		logger.Error("media watch failed", "error", err)
	}
}
