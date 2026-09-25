package open

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/afterwait"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

type Exec struct {
	Path []cmd.WorkDirArg `short:"p" long:"path" help:"Prepend directories to PATH (can be specified multiple times)"`
	sep  cmd.Dash
	cmd  []cmd.StringArg
}

func (Exec) Description() string {
	return `Execute a command using the platform-appropriate exec driver

Examples:
  modot open exec -- git status
  modot open exec -- ls -la /data
  modot open exec -- command --with-flags
  modot open exec --path /custom/bin -- mycommand`
}

func (e *Exec) Run(ctx context.Context) error {
	args := cmd.Values(e.cmd)
	if len(args) == 0 {
		return cmd.ErrUsage
	}

	command, err := execdriver.Run(ctx, args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	pathDirs := cmd.Values(e.Path)
	if len(pathDirs) > 0 {
		env := os.Environ()
		if command.Env != nil {
			env = command.Env
		}

		pathModified := false
		for i, envVar := range env {
			if after, ok := strings.CutPrefix(envVar, "PATH="); ok {
				currentPath := after

				newPaths := make([]string, 0, len(pathDirs)+1)
				for _, dir := range pathDirs {
					absDir, err := filepath.Abs(dir)
					if err != nil {
						return fmt.Errorf("invalid path %q: %w", dir, err)
					}
					newPaths = append(newPaths, absDir)
				}
				newPaths = append(newPaths, currentPath)

				env[i] = "PATH=" + strings.Join(newPaths, string(os.PathListSeparator))
				pathModified = true
				break
			}
		}

		if !pathModified {
			newPaths := make([]string, 0, len(pathDirs))
			for _, dir := range pathDirs {
				absDir, err := filepath.Abs(dir)
				if err != nil {
					return fmt.Errorf("invalid path %q: %w", dir, err)
				}
				newPaths = append(newPaths, absDir)
			}
			env = append(env, "PATH="+strings.Join(newPaths, string(os.PathListSeparator)))
		}

		command.Env = env
	}

	afterwait.Exec(ctx, command)
	return nil
}
