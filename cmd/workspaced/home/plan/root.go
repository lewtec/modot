package plan

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/workspaced/cmd/workspaced/home/apply"
	"github.com/lucasew/workspaced/internal/cmdwire"
)

type Command struct {
	ShowNoop cmd.Flag      `long:"show-noop" help:"Also show files that would not change"`
	Prefix   cmd.StringArg `long:"prefix" default:"~" help:"directory that receives home files"`
}

func (Command) Description() string {
	return "Show what would be applied (dry-run)"
}

func (c *Command) Run(ctx context.Context) error {
	prefix := c.Prefix.Value()
	return cmdwire.RunAfterWait(ctx, true, c.ShowNoop.Value(), func(ctx context.Context, dryRun, showNoop bool) func() error {
		return apply.Schedule(ctx, dryRun, showNoop, prefix)
	})
}
