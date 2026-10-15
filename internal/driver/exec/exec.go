package exec

import (
	"context"
	"log/slog"
	"os/exec"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	lewexec "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/executil"
)

// IsBinaryAvailable checks if a command exists in PATH using the selected driver.
func IsBinaryAvailable(ctx context.Context, name string) bool {
	return lewexec.IsBinaryAvailable(ctx, name)
}

// Run creates an exec.Cmd using the selected driver.
// Stderr defaults to a session live-row writer when a Session is on ctx,
// otherwise the process os.Stderr. Context writers from executil override
// stdout/stderr when set.
func Run(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	slog.Debug("running command", "name", name, "args", args)
	d, err := lewdriver.Get[lewexec.Driver](ctx)
	if err != nil {
		return nil, err
	}
	// Command has no context. Bind the caller's ctx so cancel still kills
	// the process when the caller runs the returned Cmd itself.
	base := d.Command(name, args...)
	path := base.Path
	if path == "" {
		path = name
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = base.Env
	cmd.Dir = base.Dir
	attachDefaultWriters(ctx, cmd)
	return cmd, nil
}

// Which locates a command in PATH using the selected driver.
func Which(ctx context.Context, name string) (string, error) {
	return lewexec.Which(ctx, name)
}

// MustRun creates and returns an exec.Cmd using the selected driver.
// If the driver cannot be loaded, it logs a warning (name + error) and falls
// back to os/exec.CommandContext so callers that expect *exec.Cmd keep working.
// Prefer Run when the caller can handle the error.
func MustRun(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd, err := Run(ctx, name, args...)
	if err != nil {
		slog.Warn("exec driver unavailable; falling back to raw os/exec", "name", name, "error", err)
		cmd = exec.CommandContext(ctx, name, args...) //nolint:forbidigo // facade fallback when driver.Get fails
		attachDefaultWriters(ctx, cmd)
		return cmd
	}
	return cmd
}

// attachDefaultWriters sets stderr to a session live-row writer when a Session
// is on ctx (CR / in-line CSI stay on one row until newline). Otherwise stderr
// is the process os.Stderr. executil context overrides still win.
// Stdout is left unset unless the context carries one, so callers can still
// use Cmd.Output() or assign a capture buffer.
func attachDefaultWriters(ctx context.Context, cmd *exec.Cmd) {
	cmd.Stderr = taskgroup.LineWriterFrom(ctx)
	if stdout := executil.Stdout(ctx); stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr := executil.Stderr(ctx); stderr != nil {
		cmd.Stderr = stderr
	}
}

// RequireBinary returns lewdriver.ErrIncompatible when name is missing from PATH.
func RequireBinary(ctx context.Context, name string) error {
	return lewexec.RequireBinary(ctx, name)
}
