package codebase

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/workspaced/internal/cmdwire"
)

type Plan struct {
	ShowNoop cmd.Flag `long:"show-noop" help:"Also show files that would not change"`
	Prefix   Prefix   `long:"prefix" ctx:"prefix" help:"directory that receives codebase files"`
}

func (Plan) Description() string {
	return "Show what would be applied to the repo root (dry-run)"
}

func (p *Plan) Run(ctx context.Context) error {
	return cmdwire.RunAfterWait(ctx, true, p.ShowNoop.Value(), Schedule)
}
