package notification

import lewnotify "github.com/lewtec/lewkit/x/driver/notification"

const (
	NixBuildNotificationID uint32 = 101
	BackupNotificationID   uint32 = 102
)

type Notification = lewnotify.Notification
type Driver = lewnotify.Driver
