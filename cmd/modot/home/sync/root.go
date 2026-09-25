package sync

import (
	"context"
	"fmt"

	envdriver "github.com/lewtec/modot/internal/driver/env"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	"github.com/lewtec/modot/internal/logging"
)

type Command struct{}

func (Command) Description() string {
	return "Pull dotfiles changes and apply them"
}

func (c *Command) Run(ctx context.Context) error {
	root, err := envdriver.GetDotfilesRoot(ctx)
	if err != nil {
		return fmt.Errorf("get dotfiles root: %w", err)
	}

	logger := logging.GetLogger(ctx)
	logger.Info("==> Pulling dotfiles changes...")
	pullCmd := execdriver.MustRun(ctx, "git", "-C", root, "pull")
	pullCmd.Stdout = pullCmd.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	cmd := execdriver.MustRun(ctx, "modot", "self-update")
	cmd.Stdout = cmd.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}
