package open

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/afterwait"
	"github.com/lewtec/modot/internal/tool"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

type Lazy struct {
	Bin  cmd.StringArg `long:"bin" help:"Binary name to resolve inside the tool package" default:""`
	Home cmd.Flag      `long:"home" help:"Resolve the lazy tool using the home/dotfiles workspace"`
	tool cmd.StringArg
	sep  cmd.Dash
	args []cmd.StringArg
}

func (Lazy) Description() string {
	return "Run a lazy tool resolved from home config and modot.lock.json"
}

func (l *Lazy) Run(ctx context.Context) error {
	toolName := l.tool.Value()
	return runLazyTool(ctx, l.Home.Value(), toolName, l.binName(toolName), cmd.Values(l.args))
}

func (l *Lazy) binName(toolName string) string {
	if v := l.Bin.Value(); v != "" {
		return v
	}
	return toolName
}

func runLazyTool(ctx context.Context, homeMode bool, toolName, binName string, toolArgs []string) error {
	resolver := tool.ResolveLazyTool
	if homeMode {
		resolver = tool.ResolveHomeLazyTool
	}

	taskgroup.Go(ctx, "open:lazy:"+toolName, taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("resolving " + toolName)
		binPath, err := resolver(ctx, toolName, binName)
		if err != nil {
			return err
		}
		execCtx := context.WithoutCancel(ctx)
		c, err := execdriver.Run(execCtx, binPath, toolArgs...)
		if err != nil {
			return fmt.Errorf("create command: %w", err)
		}
		afterwait.Exec(ctx, c)
		return nil
	})
	return nil
}
