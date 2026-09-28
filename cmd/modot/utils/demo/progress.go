package demo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	lewnotify "github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/modot/internal/driver/notification"
)

type Progress struct{}

func (Progress) Description() string { return "Demo progress notification" }

func (*Progress) Run(ctx context.Context) error {
	n := &notification.Notification{
		Title: "Progress Demo",
		Icon:  "utilities-terminal",
	}
	for i := 1; i <= 10; i++ {
		percent := i * 10
		n.Message = fmt.Sprintf("Step %d of 10...", i)
		n.HasProgress = true
		n.ID = 69
		n.Progress = float64(percent) / 100.0
		if err := lewnotify.Notify(ctx, *n); err != nil {
			slog.Error("error sending progress notification", "error", err)
		}
		time.Sleep(time.Second)
	}
	n.Message = "Demo complete!"
	n.Progress = 1.0
	if err := lewnotify.Notify(ctx, *n); err != nil {
		slog.Error("error sending final notification", "error", err)
	}
	return nil
}
