package selfbin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	envdriver "github.com/lewtec/modot/internal/driver/env"
	"github.com/lewtec/modot/internal/driver/shim"
	"github.com/lewtec/modot/internal/logging"
)

// InstallPaths returns the fixed bin dir and modot binary path under the
// env driver's user data dir (ResolveHomeDir fallback during bootstrap).
func InstallPaths(ctx context.Context) (installDir, installPath string, err error) {
	dataDir, err := envdriver.GetUserDataDir(ctx)
	if err != nil {
		home, homeErr := envdriver.ResolveHomeDir()
		if homeErr != nil {
			return "", "", fmt.Errorf("get home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share", "modot")
		if mkErr := os.MkdirAll(dataDir, 0o755); mkErr != nil {
			return "", "", fmt.Errorf("create user data dir: %w", mkErr)
		}
	}
	installDir = filepath.Join(dataDir, "bin")
	name := "modot"
	if runtime.GOOS == "windows" {
		name = "modot.exe"
	}
	installPath = filepath.Join(installDir, name)
	return installDir, installPath, nil
}

// EnsureModotShim writes ~/.local/bin/modot → modotPath.
func EnsureModotShim(ctx context.Context, modotPath string) error {
	shimPath, err := shim.GenerateInLocalBin(ctx, "modot", []string{modotPath})
	if err != nil {
		return err
	}
	logging.GetLogger(ctx).Info("modot shim ready", "path", shimPath, "target", modotPath)
	return nil
}
