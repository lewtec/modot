package termux

import (
	"context"
	"fmt"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	kitnotify "github.com/lewtec/lewkit/x/driver/notification"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

func init() {
	lewdriver.Register[kitnotify.Driver](factory{})
}

type factory struct{}

func (factory) ID() string   { return "notification_termux" }
func (factory) Name() string { return "Termux" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireBinary(ctx, "termux-notification")
}

func (factory) New(context.Context) (kitnotify.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Notify(ctx context.Context, n kitnotify.Notification) error {
	args := []string{
		"--title", n.Title,
		"--content", n.Message,
	}
	if n.ID != 0 {
		args = append(args, "--id", fmt.Sprintf("%d", n.ID))
	}
	return execdriver.MustRun(ctx, "termux-notification", args...).Run()
}
