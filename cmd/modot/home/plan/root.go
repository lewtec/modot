package plan

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/cmd/modot/home/apply"
	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/cmdwire"
)

type Command struct {
	ShowNoop cmd.Flag      `long:"show-noop" help:"Also show files that would not change"`
	Prefix   cmdarg.Prefix `long:"prefix" ctx:"prefix" default:"~" help:"directory that receives home files"`
}

func (Command) Description() string {
	return "Show what would be applied (dry-run)"
}

func (c *Command) Run(ctx context.Context) error {
	return cmdwire.RunAfterWait(ctx, true, c.ShowNoop.Value(), apply.Schedule)
}
