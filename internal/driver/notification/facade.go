package notification

import (
	"context"

	kitnotify "github.com/lewtec/lewkit/x/driver/notification"
)

func Notify(ctx context.Context, n *Notification) error {
	if n == nil {
		return kitnotify.Notify(ctx, Notification{})
	}
	return kitnotify.Notify(ctx, *n)
}
