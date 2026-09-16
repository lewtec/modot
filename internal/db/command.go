package db

import "context"

// Command is the reusable --database group. Inline it with
//
//	*db.Command `flatten:""`
//
// so --database lands on the parent. The same type can be a real
// subcommand later (`*db.Command` without flatten).
type Command struct {
	Database Arg `long:"database" help:"sqlite URL" ctx:""`
}

func (Command) Description() string {
	return "SQLite store"
}

func (c *Command) Open(ctx context.Context) (*DB, error) {
	if c == nil {
		return Open(ctx)
	}
	return OpenArg(ctx, c.Database)
}
