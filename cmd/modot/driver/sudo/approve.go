package sudo

import (
	"context"
	"log/slog"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	"github.com/lewtec/modot/internal/sudo"
)

type Approve struct {
	slug cmd.StringArg
}

func (Approve) Description() string { return "Approve and execute a pending command" }

func (c *Approve) Run(ctx context.Context) error {
	slug := c.slug.Value()
	sc, err := sudo.Get(slug)
	if err != nil {
		return err
	}

	slog.Info("approving command", "command", sc.Command, "args", sc.Args, "slug", slug)
	defer func() {
		if err := func() error { return sudo.Remove(slug) }(); err != nil {
			slog.Error("unexpected error", "op", "sudo-remove", "error", err)
		}
	}()

	ec, err := execdriver.Run(ctx, "sudo", append([]string{"-E", sc.Command}, sc.Args...)...)
	if err != nil {
		return err
	}
	ec.Stdout = ec.Stderr
	ec.Stdin = os.Stdin
	ec.Dir = sc.Cwd
	ec.Env = sc.Env

	return ec.Run()
}
