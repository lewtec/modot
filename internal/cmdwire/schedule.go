package cmdwire

import (
	"context"

	"github.com/lucasew/workspaced/internal/cmdctx"
	"github.com/lucasew/workspaced/pkg/taskgroup"
)

// ScheduleFunc wires plan/apply work into a task group and returns an AfterWait
// report printer.
type ScheduleFunc func(g *taskgroup.Group, ctx context.Context, dryRun, showNoop bool) func() error

// RunAfterWait is the shared plan/apply Run body: optionally force dry-run
// (plan), schedule work, print the report after session wait.
func RunAfterWait(ctx context.Context, forceDryRun, showNoop bool, schedule ScheduleFunc) error {
	dryRun := forceDryRun || cmdctx.IsDryRun(ctx)

	if sess := taskgroup.SessionFrom(ctx); sess != nil {
		sess.Overlay(ctx)
	}
	if forceDryRun {
		ctx = cmdctx.WithDryRun(ctx, true)
		if sess := taskgroup.SessionFrom(ctx); sess != nil {
			sess.Overlay(ctx)
		}
		g := taskgroup.MustFromContext(ctx)
		taskgroup.MustSessionFrom(ctx).AfterWait(schedule(g, ctx, true, showNoop))
		return nil
	}

	g := taskgroup.MustFromContext(ctx)
	taskgroup.MustSessionFrom(ctx).AfterWait(schedule(g, ctx, dryRun, showNoop))
	return nil
}
