package input

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

type Launch struct{}

func (Launch) Description() string { return "Application launcher" }

func (*Launch) Run(ctx context.Context) error {
	return launcher.RunApp(ctx)
}

type Window struct{}

func (Window) Description() string { return "Window switcher" }

func (*Window) Run(ctx context.Context) error {
	return launcher.SwitchWindow(ctx)
}
