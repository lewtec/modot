package codebase

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/modot/internal/lsp"
	"github.com/lewtec/modot/internal/logging"
)

type Lsp struct{}

func (Lsp) Description() string {
	return "Experimental: language server router (stdio LSP proxy driven by modot.cue)"
}

func (*Lsp) Run(ctx context.Context) error {
	logger := logging.GetLogger(ctx)
	logger.Info("codebase lsp starting (stdio)")
	err := lsp.Run(ctx, os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("lsp: %w", err)
	}
	return nil
}
