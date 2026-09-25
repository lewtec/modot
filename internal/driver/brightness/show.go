package brightness

import (
	"context"

	kitbrightness "github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/lewtec/modot/internal/driver/notification"
)

func ShowStatus(ctx context.Context) error {
	status, err := kitbrightness.Status(ctx)
	if err != nil {
		return err
	}

	n := notification.Notification{
		ID:          notification.StatusNotificationID,
		Title:       "Brightness",
		Message:     status.Name,
		Icon:        "display-brightness",
		Progress:    status.Brightness,
		HasProgress: true,
	}
	return notification.Notify(ctx, &n)
}
