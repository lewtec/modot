package tray

import (
	"context"
	lewdriver "github.com/lewtec/lewkit/x/driver"
)

// GetDefault returns the appropriate tray driver for the current environment.
func GetDefault(ctx context.Context) (Driver, error) {
	return lewdriver.Get[Driver](ctx)
}
