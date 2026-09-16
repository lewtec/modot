package tool

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/internal/tool"
)

type Install struct {
	spec cmd.StringArg
}

func (Install) Description() string { return "Install a tool" }

func (i *Install) Run(ctx context.Context) error {
	manager, err := tool.NewManager()
	if err != nil {
		return err
	}

	spec := i.spec.Value()
	taskgroup.Go(ctx, "tool:install:"+spec, taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("installing " + spec)
		return manager.Install(ctx, spec)
	})
	return nil
}
