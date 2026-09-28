package media

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	lewmedia "github.com/lewtec/lewkit/x/driver/media"
	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
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
	note, ok := lewmedia.StatusNotification(meta)
	if !ok {
		logger := slog.Default()
		logger.Warn("no active player with title found")
		return nil
	}
	iconPath := ""
	if meta.ArtUrl != "" {
		var err error
		iconPath, err = GetArtCachePath(ctx, meta.ArtUrl)
		if err != nil {
			if err != nil {
				slog.ErrorContext(ctx, "unexpected error", "error", err)
			}
		}
	}
	note.Icon = iconPath

	logger := slog.Default()
	logger.Info("sending media notification",
		"player", meta.Player,
		"title", note.Title,
		"artist", note.Message,
		"progress", note.Progress,
		"icon", iconPath,
	)

	return lewnotify.Notify(ctx, note)
}

func Watch(ctx context.Context) {
	err := lewmedia.Watch(ctx, func(meta *Metadata) {
		if err := Notify(ctx, meta); err != nil {
			if err != nil {
				slog.ErrorContext(ctx, "unexpected error", "error", err)
			}
		}
	})
	if err != nil {
		logger := slog.Default()
		logger.Error("media watch failed", "error", err)
	}
}
