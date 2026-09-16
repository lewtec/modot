package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/pkg/logging"
)

type Loop struct{}

func (Loop) Description() string {
	return `Demo a 5-iteration loop (schedule on the session from context and return)

Uses the same primitives as the other demos:
- taskgroup.Go(ctx, "loop-demo", ..., func(ctx, s) { ... s.Update/Progress })
- progress UI comes from the root session (TERM=dumb / CI / non-tty stay plain)`
}

func (*Loop) Run(ctx context.Context) error {
	taskgroup.Go(ctx, "loop-demo", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		logger := logging.GetLogger(ctx)

		for i := 1; i <= 5; i++ {
			time.Sleep(1 * time.Second)
			logger.Info("log line from loop", "iteration", i)
			s.Update(fmt.Sprintf("step %d/5", i))
			s.Progress(int64(i), 5)
		}
		return nil
	})
	return nil
}
