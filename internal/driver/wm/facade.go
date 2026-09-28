package wm

import (
	"context"
	"log/slog"

	lewwm "github.com/lewtec/lewkit/x/driver/wm"
	"github.com/lewtec/modot/internal/driver/media"
)

// ToggleScratchpadWithInfo toggles the scratchpad and shows a media status notification.
func ToggleScratchpadWithInfo(ctx context.Context) error {
	if err := lewwm.ToggleScratchpad(ctx); err != nil {
		return err
	}
	if err := media.ShowStatus(ctx); err != nil {
		if err != nil {
			slog.ErrorContext(ctx, "unexpected error", "error",

				// NextWorkspace switches to the next numbered workspace, moving the focused container when move is set.
				err)
		}
	}
	return nil
}

func NextWorkspace(ctx context.Context, move bool) error {
	name, err := lewwm.AdvanceWorkspace()
	if err != nil {
		return err
	}
	return lewwm.SwitchToWorkspace(ctx, name, move)
}
