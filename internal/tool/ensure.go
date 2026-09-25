// Package tool ensures a binary in the modot tool store.
// The install itself is github.com/lewtec/lewkit/x/tool.
package tool

import (
	"context"

	lewtool "github.com/lewtec/lewkit/x/tool"
)

// EnsureInstalled ensures toolSpec is on disk and returns the absolute path
// of binary. Same path as `modot tool which`.
func EnsureInstalled(ctx context.Context, toolSpec, binary string) (string, error) {
	dir, err := GetToolsDir()
	if err != nil {
		return "", err
	}
	store, err := lewtool.Open(dir)
	if err != nil {
		return "", err
	}
	return store.Ensure(WithCmdFlags(ctx), toolSpec, binary)
}
