package codebase

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	lewgit "github.com/lewtec/lewkit/x/git"
	"github.com/lewtec/modot/internal/tool"
)

type CIStatus struct {
	args []cmd.StringArg
}

func (CIStatus) Description() string {
	return "Run ci-status from the workspace lazy_tools pin"
}

func (c *CIStatus) Run(ctx context.Context) error {
	run, err := tool.EnsureAndRunLazy(ctx, "ci_status", "ci-status", cmd.Values(c.args)...)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	info, ok := (&lewgit.Git{}).Info(ctx, wd)
	if !ok || info.Toplevel == "" {
		return fmt.Errorf("find git root: %s", wd)
	}
	run.Dir = info.Toplevel
	run.Stdin = os.Stdin
	run.Stdout = run.Stderr
	return run.Run()
}
