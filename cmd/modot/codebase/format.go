package codebase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lewtec/lewkit/x/cmd"
	lewgit "github.com/lewtec/lewkit/x/git"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/checks/formatter"
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

	info, ok := (&lewgit.Git{}).Info(ctx, absPath)
	if !ok || info.Toplevel == "" {
		return fmt.Errorf("find git root (format must run inside a git repo): %s", absPath)
	}
	root := info.Toplevel

	taskgroup.Go(ctx, "codebase:format", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("running formatters")
		return formatter.RunAll(ctx, root)
	})
	return nil
}
