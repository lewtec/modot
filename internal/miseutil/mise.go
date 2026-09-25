package miseutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/modot/internal/atomicfile"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	"github.com/lewtec/modot/internal/driver/shim/bash"
	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/tool"
)

var (
	// ErrBinaryNotFound is returned when a binary is not found in a mise tool install tree.
	ErrBinaryNotFound = errors.New("binary not found")
)

// Ensure returns a path to the mise CLI via lazy_tools.mise (registry:mise).
// Version comes from the workspace lockfile. Same path as
// `modot open lazy mise`.
func Ensure(ctx context.Context) (string, error) {
	return tool.ResolveLazyTool(ctx, "mise", "mise")
}

// Output runs the mise CLI with args and returns combined stdout.
// Used by the mise: package backend (not for installing mise itself).
func Output(ctx context.Context, args ...string) ([]byte, error) {
	misePath, err := Ensure(ctx)
	if err != nil {
		return nil, err
	}
	cmd, err := execdriver.Run(ctx, misePath, args...)
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

// Run runs the mise CLI with args, wiring stdio to the process.
func Run(ctx context.Context, args ...string) error {
	misePath, err := Ensure(ctx)
	if err != nil {
		return err
	}
	cmd, err := execdriver.Run(ctx, misePath, args...)
	if err != nil {
		return err
	}
	cmd.Stdout = cmd.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Latest resolves the latest version of a mise package spec (e.g. "node").
func Latest(ctx context.Context, spec string) (string, error) {
	return trimmedOutput(ctx, "latest", spec)
}

// Where returns the install root for a mise package spec.
func Where(ctx context.Context, toolSpec string) (string, error) {
	return trimmedOutput(ctx, "where", toolSpec)
}

func trimmedOutput(ctx context.Context, args ...string) (string, error) {
	out, err := Output(ctx, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ResolveBinPath finds binName under the install root of a mise package spec.
func ResolveBinPath(ctx context.Context, binName, toolSpec string) (string, error) {
	root, err := Where(ctx, toolSpec)
	if err != nil {
		return "", err
	}

	if binPath := lewtool.FindBinary(root, binName); binPath != "" {
		return binPath, nil
	}

	return "", fmt.Errorf("%w: %q under %s", ErrBinaryNotFound, binName, root)
}

// EnsureLocalBinWrapper writes ~/.local/bin/mise so PATH users re-enter the
// standard lazy route (open lazy --home mise). Integration only — not a
// separate install path for the binary.
//
// modotBin is the absolute path to the modot binary; when empty,
// the default under the user data dir is used.
func EnsureLocalBinWrapper(ctx context.Context, modotBin string) error {
	logger := logging.GetLogger(ctx)
	home, err := envdriver.ResolveHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	if strings.TrimSpace(modotBin) == "" {
		dataDir, derr := envdriver.GetUserDataDir(ctx)
		if derr != nil {
			dataDir = filepath.Join(home, ".local", "share", "modot")
		}
		modotBin = filepath.Join(dataDir, "bin", "modot")
	}

	wrapperDir := filepath.Join(home, ".local", "bin")
	wrapperPath := filepath.Join(wrapperDir, "mise")
	shell := bash.GetShell(ctx)
	// Same argv shape as modules/mise and open lazy --home.
	expectedContent := fmt.Sprintf(
		"#!%s\nexec -a \"$0\" %s open lazy --home --bin mise mise -- \"$@\"\n",
		shell, modotBin,
	)

	if content, err := os.ReadFile(wrapperPath); err == nil && string(content) == expectedContent {
		return nil
	}

	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		return fmt.Errorf("create wrapper directory: %w", err)
	}
	if err := atomicfile.WriteString(wrapperPath, expectedContent, 0o755); err != nil {
		return fmt.Errorf("write mise wrapper: %w", err)
	}

	logger.Info("created mise wrapper", "path", wrapperPath, "modot", modotBin)
	return nil
}
