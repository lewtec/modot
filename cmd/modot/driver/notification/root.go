package notification

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/driver/notification"
)

type Command struct {
	Title    cmd.StringArg               `short:"t" long:"title" help:"Notification title" default:"Modot"`
	Message  cmd.StringArg               `short:"m" long:"message" help:"Notification message"`
	Icon     cmd.StringArg               `short:"i" long:"icon" help:"Notification icon"`
	Urgency  cmd.EnumArg[cmdarg.Urgency] `short:"u" long:"urgency" help:"Notification urgency (low, normal, critical)" default:"normal"`
	Progress cmd.FloatArg[float64]       `short:"p" long:"progress" help:"Notification progress (0.0-1.0)"`
}

func (Command) Description() string {
	return "Send a desktop notification"
}

func (c *Command) Run(ctx context.Context) error {
	n := &notification.Notification{
		Title:    c.Title.Value(),
		Message:  c.Message.Value(),
		Icon:     c.Icon.Value(),
		Urgency:  c.Urgency.Value().String(),
		Progress: c.Progress.Value(),
	}
	return notification.Notify(ctx, n)
}
