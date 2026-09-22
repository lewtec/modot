package tool

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/lucasew/workspaced/internal/tool"
)

type Install struct {
	spec cmd.StringArg
}

func (Install) Description() string { return "Install a tool" }

func (i *Install) Run(ctx context.Context) error {
	dir, err := tool.GetToolsDir()
	if err != nil {
		return err
	}
	store, err := lewtool.Open(dir)
	if err != nil {
		return err
	}

	spec := i.spec.Value()
	taskgroup.Go(ctx, "tool:install:"+spec, taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("installing " + spec)
		return store.Install(tool.WithCmdFlags(ctx), spec)
	})
	return nil
}
