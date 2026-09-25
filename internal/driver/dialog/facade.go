package dialog

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

func Choose(ctx context.Context, opts ChooseOptions) (*Item, error) {
	return launcher.Choose(ctx, opts)
}

func Prompt(ctx context.Context, prompt string) (string, error) {
	return launcher.Prompt(ctx, prompt)
}

func Confirm(ctx context.Context, message string) (bool, error) {
	return launcher.Confirm(ctx, message)
}

func RunApp(ctx context.Context) error {
	return launcher.RunApp(ctx)
}

func SwitchWindow(ctx context.Context) error {
	return launcher.SwitchWindow(ctx)
}
