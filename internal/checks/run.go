package checks

import (
	"os/exec"
)

// RunAttached runs cmd in dir with stdout sharing the session live-row writer
// already attached to stderr by exec.Run.
func RunAttached(cmd *exec.Cmd, dir string) error {
	cmd.Dir = dir
	if cmd.Stderr != nil {
		cmd.Stdout = cmd.Stderr
	}
	return cmd.Run()
}
