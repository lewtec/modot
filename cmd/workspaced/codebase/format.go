package codebase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/internal/checks/formatter"
	"github.com/lucasew/workspaced/internal/git"
	"github.com/lucasew/workspaced/internal/taskui"
)

type Format struct {
	path cmd.WorkDirArg
}

func (Format) Description() string {
	return "Format code in the repository (runs at git root)"
}

func (f *Format) Run(ctx context.Context) error {
	absPath, err := filepath.Abs(f.path.Value())
	if err != nil {
		return err
	}

	root, err := git.GetRoot(ctx, absPath)
	if err != nil {
		return fmt.Errorf("find git root (format must run inside a git repo): %w", err)
	}

	return taskui.Run(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, "codebase:format", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("running formatters")
			return formatter.RunAll(ctx, root)
		})
		return nil
	})
}
