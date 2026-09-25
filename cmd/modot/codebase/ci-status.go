package codebase

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/git"
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
	run.Dir, err = git.GetRoot(ctx, wd)
	if err != nil {
		return err
	}
	run.Stdin = os.Stdin
	run.Stdout = run.Stderr
	return run.Run()
}
