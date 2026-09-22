// Package tool chooses the workspaced tool directory, shims, and lock rows.
// Versioned installs are github.com/lewtec/lewkit/x/tool.
package tool

import (
	"context"
	"path/filepath"

	lewtool "github.com/lewtec/lewkit/x/tool"

	"github.com/lucasew/workspaced/internal/cmdctx"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
)

func GetToolsDir() (string, error) {
	return workspacedShareDir("tools")
}

func GetShimsDir() (string, error) {
	return workspacedShareDir("shims")
}

func workspacedShareDir(leaf string) (string, error) {
	home, err := envdriver.ResolveHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "workspaced", leaf), nil
}

// WithCmdFlags copies --no-cache and --dry-run onto ctx for lewtool.Store.
// Plan can flip dry-run after setup, so this is read at the call, not at startup.
func WithCmdFlags(ctx context.Context) context.Context {
	if cmdctx.IsNoCache(ctx) {
		ctx = lewtool.WithNoCache(ctx)
	}
	if cmdctx.IsDryRun(ctx) {
		ctx = lewtool.WithDryRun(ctx)
	}
	return ctx
}
