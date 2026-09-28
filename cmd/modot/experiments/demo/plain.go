package demo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type Plain struct{}

func (Plain) Description() string {
	return `Run tasks under the root session; observe plain-style rendering behavior

Schedules the same kind of work as the default demo. Set TERM=dumb
(or CI=1 or NO_COLOR) to skip the progress TUI and see slog only.`
}

func (*Plain) Run(ctx context.Context) error {
	slog.Info("Scheduling on the session from context.")
	slog.Info("Pipe the command or set TERM=dumb/CI=1/NO_COLOR to observe plain behavior on tty.")

	fetch := taskgroup.Go(ctx, "fetch", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("contacting API")
		time.Sleep(80 * time.Millisecond)
		slog.Info("http response", "status", "200 OK")
		s.Progress(0, 4)
		for i := 1; i <= 4; i++ {
			s.Progress(int64(i), 4)
			s.Update(fmt.Sprintf("page %d", i))
			time.Sleep(90 * time.Millisecond)
		}
		return nil
	})

	process := taskgroup.Go(ctx, "process", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("crunching numbers")
		for i := range 3 {
			slog.Info("batch processed", "num", i)
			time.Sleep(110 * time.Millisecond)
		}
		return nil
	}, fetch)

	taskgroup.Go(ctx, "write", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("writing artifacts")
		time.Sleep(150 * time.Millisecond)
		slog.Info("fsync complete")
		return nil
	}, process)

	return nil
}
