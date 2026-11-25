package demo

import (
	"context"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/logging"
)

type Nested struct{}

func (Nested) Description() string {
	return `Demonstrate Isolate as an error-boundary with child tasks

Isolate runs work under an unnamed node that shares pools with the parent but
does not cancel parent siblings on failure and does not add its own progress bar.

Schedule named tasks inside the isolated ctx (or use GoIsolated / Map). Prefer
Map when you need aggregate progress.`
}

func (*Nested) Run(ctx context.Context) error {
	logger := logging.GetLogger(ctx)
	logger.Info("scheduling bundle with Isolate children")

	taskgroup.Go(ctx, "bundle", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("starting bundle phase")
		time.Sleep(60 * time.Millisecond)
		err := taskgroup.Isolate(ctx, func(ctx context.Context) error {
			icons := taskgroup.Go(ctx, "bundle:icons", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				logger := logging.GetLogger(ctx)
				s.Update("generating icons")
				for i := 0; i < 3; i++ {
					logger.Info("icon", "num", i)
					time.Sleep(90 * time.Millisecond)
				}
				return nil
			})
			taskgroup.Go(ctx, "bundle:manifest", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
				logger := logging.GetLogger(ctx)
				s.Update("writing manifest.json")
				time.Sleep(130 * time.Millisecond)
				logger.Info("manifest written")
				return nil
			}, icons)
			return nil
		})
		if err != nil {
			return err
		}
		s.Update("bundle complete")
		logging.GetLogger(ctx).Info("isolate subtree done")
		return nil
	})
	return nil
}
