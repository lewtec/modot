package notification

import kitnotify "github.com/lewtec/lewkit/x/driver/notification"

const (
	StatusNotificationID   uint32 = 100
	NixBuildNotificationID uint32 = 101
	BackupNotificationID   uint32 = 102
)

type Notification = kitnotify.Notification
type Driver = kitnotify.Driver
