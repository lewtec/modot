package executil

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/pkg/logging"
)

type stdoutKey struct{}
type stderrKey struct{}
type envKey struct{}

// WithStdout returns a context carrying an io.Writer for standard output.
func WithStdout(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, stdoutKey{}, w)
}

// Stdout retrieves the stdout writer from the context, or nil if not set.
func Stdout(ctx context.Context) io.Writer {
	w, ok := ctx.Value(stdoutKey{}).(io.Writer)
	if !ok {
		return nil
	}
	return w
}

// WithStderr returns a context carrying an io.Writer for standard error.
func WithStderr(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, stderrKey{}, w)
}

// Stderr retrieves the stderr writer from the context, or nil if not set.
func Stderr(ctx context.Context) io.Writer {
	w, ok := ctx.Value(stderrKey{}).(io.Writer)
	if !ok {
		return nil
	}
	return w
}

// WithEnv returns a context carrying environment variables as a slice of "KEY=VALUE" strings.
func WithEnv(ctx context.Context, env []string) context.Context {
	return context.WithValue(ctx, envKey{}, env)
}

// Env retrieves the environment variable slice from the context, or nil if not set.
func Env(ctx context.Context) []string {
	env, ok := ctx.Value(envKey{}).([]string)
	if !ok {
		return nil
	}
	return env
}

// InheritContextWriters sets cmd.Stdout and cmd.Stderr from context writers when
// present. Otherwise stderr stays on the session live-row writer (or finished
// lines on os.Stderr with no session), and stdout shares that writer so child
// chatter does not inherit the process tty and walk the progress overlay.
// Context writers still win (daemon packet streams).
func InheritContextWriters(ctx context.Context, cmd *exec.Cmd) {
	if w := Stderr(ctx); w != nil {
		cmd.Stderr = w
	} else if cmd.Stderr == nil {
		cmd.Stderr = taskgroup.LineWriterFrom(ctx)
	}
	if w := Stdout(ctx); w != nil {
		cmd.Stdout = w
	} else if cmd.Stdout == nil {
		cmd.Stdout = cmd.Stderr
	}
}

// StdoutOr returns the context stdout writer, or fallback when unset.
func StdoutOr(ctx context.Context, fallback io.Writer) io.Writer {
	if w := Stdout(ctx); w != nil {
		return w
	}
	return fallback
}

// StderrOr returns the context stderr writer, or fallback when unset.
func StderrOr(ctx context.Context, fallback io.Writer) io.Writer {
	if w := Stderr(ctx); w != nil {
		return w
	}
	return fallback
}

// GetEnv retrieves an environment variable from the context or the system.
func GetEnv(ctx context.Context, key string) string {
	if env := Env(ctx); env != nil {
		for _, e := range env {
			if strings.HasPrefix(e, key+"=") {
				return e[len(key)+1:]
			}
		}
	}
	return os.Getenv(key)
}

// GetBinaryHash returns the SHA256 hash of the current executable
func GetBinaryHash(ctx context.Context) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("executable path: %w", err)
	}

	file, err := os.Open(exePath)
	if err != nil {
		return "", fmt.Errorf("open executable: %w", err)
	}
	defer logging.Close(ctx, file)

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash executable: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetBinaryMtime returns the modification time of the current executable.
func GetBinaryMtime() (time.Time, error) {
	exePath, err := os.Executable()
	if err != nil {
		return time.Time{}, fmt.Errorf("executable path: %w", err)
	}

	info, err := os.Stat(exePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("stat executable: %w", err)
	}

	return info.ModTime(), nil
}
