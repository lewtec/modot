// Package tool ensures a binary in the workspaced tool store.
// The install itself is github.com/lewtec/lewkit/x/tool.
package tool

import (
	"context"

	lewtool "github.com/lewtec/lewkit/x/tool"

	itool "github.com/lucasew/workspaced/internal/tool"
)

// EnsureInstalled ensures toolSpec is on disk and returns the absolute path
// of binary. Same path as `workspaced tool which`.
//
// Blank-import prelude so the lewtool backends are registered.
func EnsureInstalled(ctx context.Context, toolSpec, binary string) (string, error) {
	dir, err := itool.GetToolsDir()
	if err != nil {
		return "", err
	}
	store, err := lewtool.Open(dir)
	if err != nil {
		return "", err
	}
	return store.Ensure(itool.WithCmdFlags(ctx), toolSpec, binary)
}
