package shell

import (
	"context"
	lewdriver "github.com/lewtec/lewkit/x/driver"
)

// Driver provides shell execution capabilities
type Driver interface {
	// Path returns the full path to the shell executable
	Path(ctx context.Context) (string, error)
}

// Get returns the active shell driver
func Get(ctx context.Context) (Driver, error) {
	return lewdriver.Get[Driver](ctx)
}

// Path returns the path to the active shell
func Path(ctx context.Context) (string, error) {
	return lewdriver.WithResult(ctx, func(d Driver) (string, error) { return d.Path(ctx) })
}
