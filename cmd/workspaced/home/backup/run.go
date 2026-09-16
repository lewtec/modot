package backup

import (
	"context"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/internal/backup"
)

type Run struct{}

func (Run) Description() string { return "Run full backup" }

func (*Run) Run(ctx context.Context) error {
	taskgroup.Go(ctx, "backup:run", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("running backup")
		return backup.RunFullBackup(ctx)
	})
	return nil
}
