package wm

import (
	"context"

	lewwm "github.com/lewtec/lewkit/x/driver/wm"
	"github.com/lewtec/modot/internal/driver/media"
	"github.com/lewtec/modot/internal/logging"
)

// ToggleScratchpadWithInfo toggles the scratchpad and shows a media status notification.
func ToggleScratchpadWithInfo(ctx context.Context) error {
	if err := lewwm.ToggleScratchpad(ctx); err != nil {
		return err
	}
	if err := media.ShowStatus(ctx); err != nil {
		logging.ReportError(ctx, err)
	}
	return nil
}

// NextWorkspace switches to the next numbered workspace, moving the focused container when move is set.
func NextWorkspace(ctx context.Context, move bool) error {
	name, err := lewwm.AdvanceWorkspace()
	if err != nil {
		return err
	}
	return lewwm.SwitchToWorkspace(ctx, name, move)
}
