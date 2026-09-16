package cmdwire

import (
	"context"

	"github.com/lucasew/workspaced/internal/afterwait"
	"github.com/lucasew/workspaced/internal/cmdctx"
)

// ScheduleFunc wires plan/apply work into the session and returns a report
// printer for afterwait.
type ScheduleFunc func(ctx context.Context, dryRun, showNoop bool) func() error

// RunAfterWait is the shared plan/apply Run body: optionally force dry-run
// (plan), schedule work, print the report after session wait.
func RunAfterWait(ctx context.Context, forceDryRun, showNoop bool, schedule ScheduleFunc) error {
	dryRun := forceDryRun || cmdctx.IsDryRun(ctx)
	if forceDryRun {
		cmdctx.SetDryRun(ctx, true)
		dryRun = true
	}
	afterwait.Register(ctx, schedule(ctx, dryRun, showNoop))
	return nil
}
