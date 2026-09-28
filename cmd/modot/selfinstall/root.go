package selfinstall

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/atomicfile"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	"github.com/lewtec/modot/internal/miseutil"
	"github.com/lewtec/modot/internal/selfbin"
	"github.com/lewtec/modot/internal/version"

	"github.com/lewtec/lewkit/x/cmd"
)

type Command struct {
	Force cmd.Flag `short:"f" long:"force" help:"Force reinstall (overwrite existing)"`
}

func (Command) Description() string {
	return "Install modot into tool system (bootstrap)"
}

func (c *Command) Run(ctx context.Context) error {
	taskgroup.Go(ctx, "self-install", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("self-installing modot")
		defer s.Unit()()
		return runSelfInstall(ctx, c.Force.Value())
	})
	return nil
}

func runSelfInstall(ctx context.Context, force bool) error {
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get current binary: %w", err)
	}

	// Fixed install location under real home (not Termux proot /home view).
	installDir, installPath, err := selfbin.InstallPaths(ctx)
	if err != nil {
		return err
	}

	currentVersion := version.Version()

	alreadyInstalled := false
	if !force {
		if _, err := os.Stat(installPath); err == nil {
			alreadyInstalled = true
			slog.Info("already installed", "path", installPath)
		}
	}

	// Copy binary (unless already installed and not forcing)
	if !alreadyInstalled {
		slog.Info("installing modot", "version", currentVersion, "path", installPath, "force", force)

		if err := os.MkdirAll(installDir, 0755); err != nil {
			return fmt.Errorf("create install directory: %w", err)
		}

		if err := copyFile(ctx, currentBinary, installPath); err != nil {
			return fmt.Errorf("copy binary: %w", err)
		}

		if err := os.Chmod(installPath, 0755); err != nil {
			return fmt.Errorf("set permissions: %w", err)
		}

		slog.Info("binary installed", "path", installPath)
	}

	// Always regenerate shims (even if binary already installed)
	slog.Info("regenerating shims")

	if err := selfbin.EnsureModotShim(ctx, installPath); err != nil {
		return fmt.Errorf("create shim: %w", err)
	}
	if err := createMiseShim(ctx); err != nil {
		slog.Warn("failed to create mise shim", "error", err)
	}

	slog.Info("modot installed successfully", "version", currentVersion)
	if alreadyInstalled {
		slog.Info("shims regenerated (use --force to reinstall binary)")
	}
	slog.Info("add ~/.local/bin to your PATH if not already added")

	return nil
}

func createMiseShim(ctx context.Context) error {
	// Integration shim only: re-enters the standard lazy route for mise.
	// Does not install mise; open lazy --home resolves lazy_tools.mise.
	dataDir, err := envdriver.GetUserDataDir(ctx)
	if err != nil {
		home, homeErr := envdriver.ResolveHomeDir()
		if homeErr != nil {
			return err
		}
		dataDir = filepath.Join(home, ".local", "share", "modot")
	}
	modotBin := filepath.Join(dataDir, "bin", "modot")
	if err := miseutil.EnsureLocalBinWrapper(ctx, modotBin); err != nil {
		return err
	}
	slog.Info("created mise wrapper", "target", "open lazy --home mise")
	return nil
}

func copyFile(ctx context.Context, src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closer := source; closer != nil {
			if err := closer.Close(); err != nil {
				slog.Error("unexpected error", "op", "close", "path", src, "error", err)
			}
		}
	}()

	f, err := atomicfile.Create(dst, 0o755)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Abort(); err != nil {
			slog.Error("unexpected error", "op", "atomicfile.Abort", "error", err)
		}
	}()
	if _, err := io.Copy(f, source); err != nil {
		return err
	}
	return f.CommitMode(0o755)
}
