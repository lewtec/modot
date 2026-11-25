package rsync

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/driver/rsync"
)

type Command struct {
	Exclude []cmd.StringArg `short:"e" long:"exclude" help:"Exclude pattern (repeatable)"`
	NoPerms cmd.Flag        `long:"no-perms" help:"Do not preserve permissions (like rsync --no-perms)"`
	src     cmd.StringArg
	dst     cmd.StringArg
}

func (Command) Description() string {
	return "Run a sync through the selected rsync driver (native rsync by default, gokrazy/rsync as fallback)"
}

func (c *Command) Run(ctx context.Context) error {
	opts := rsync.Options{
		Excludes:        cmd.Values(c.Exclude),
		SkipPermissions: c.NoPerms.Value(),
	}
	return rsync.Sync(ctx, c.src.Value(), c.dst.Value(), opts)
}
