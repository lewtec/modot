package open

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	kitopener "github.com/lewtec/lewkit/x/driver/opener"
)

type Command struct {
	target cmd.StringArg
}

func (Command) Description() string {
	return "Open a file or URL using the preferred opener"
}

func (c *Command) Run(ctx context.Context) error {
	return kitopener.Open(ctx, c.target.Value())
}
