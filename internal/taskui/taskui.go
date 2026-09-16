// Package taskui starts a taskgroup session and the progress view.
// Call Run from a command that schedules work. Usage/help never
// reaches here, so the TUI does not start for those paths.
package taskui

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/pkg/logging"
)

// Run enters a session (from the flatten taskgroup.Arg, or New),
// points process logs at Session.LogWriter while the TUI is up,
// then progress.Run(work).
func Run(ctx context.Context, work func(context.Context) error) error {
	s, ctx := enter(ctx)
	if progress.Interactive() {
		logging.SetProcessWriter(s.LogWriter())
		defer logging.SetProcessWriter(os.Stderr)
	}
	return progress.Run(s, ctx, work)
}

func enter(ctx context.Context) (*taskgroup.Session, context.Context) {
	if s := taskgroup.FromContext(ctx); s != nil {
		return s, ctx
	}
	base := taskgroup.DefaultLimits()
	if homeCfg, err := configcue.LoadHome(ctx); err == nil {
		base = homeCfg.ConcurrencyLimits()
	}
	if arg, ok := cmd.Lookup[taskgroup.Arg](ctx, "taskgroup"); ok {
		return arg.Enter(ctx, base)
	}
	return taskgroup.New(ctx, base)
}
