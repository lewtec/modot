package codebase

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/lewtec/modot/internal/lsp"
)

type Lsp struct{}

func (Lsp) Description() string {
	return "Experimental: language server router (stdio LSP proxy driven by modot.cue)"
}

func (*Lsp) Run(ctx context.Context) error {
	slog.Info("codebase lsp starting (stdio)")
	err := lsp.Run(ctx, os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("lsp: %w", err)
	}
	return nil
}
