// Package tool chooses the modot tool directory, shims, and lock rows.
// Versioned installs are github.com/lewtec/lewkit/x/tool.
package tool

import (
	"context"
	"path/filepath"

	lewtool "github.com/lewtec/lewkit/x/tool"

	"github.com/lewtec/modot/internal/cmdctx"
	envdriver "github.com/lewtec/modot/internal/driver/env"
)

func GetToolsDir() (string, error) {
	return modotShareDir("tools")
}

func GetShimsDir() (string, error) {
	return modotShareDir("shims")
}

func modotShareDir(leaf string) (string, error) {
	home, err := envdriver.ResolveHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "modot", leaf), nil
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
