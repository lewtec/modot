package opener

import (
	"context"
	"fmt"
	kitopener "github.com/lewtec/lewkit/x/driver/opener"
	"github.com/lewtec/modot/internal/configcue"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	"github.com/lewtec/modot/internal/executil"
	"os"
	"path/filepath"
)

// WebappConfig is used for passing parameters to OpenWebapp
type WebappConfig struct {
	URL        string
	Profile    string
	ExtraFlags []string
	Chromium   string
}

// Open opens a generic target (file or URL) using the available opener driver.
func Open(ctx context.Context, target string) error {
	return kitopener.Open(ctx, target)
}

// OpenWebapp launches a URL as a webapp using the configured browser engine.
func OpenWebapp(ctx context.Context, wa WebappConfig) error {
	cfg, err := configcue.LoadHome(ctx)
	if err != nil {
		return err
	}
	var browser struct {
		Engine string `json:"webapp"`
	}
	if err := cfg.Decode("browser", &browser); err != nil {
		return err
	}

	engine := browser.Engine
	if wa.Chromium != "" {
		engine = wa.Chromium
	}
	args := []string{}
	if wa.URL != "" {
		args = append(args, "--app="+envdriver.NormalizeURL(wa.URL))
	}

	if wa.Profile != "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		profileDir := filepath.Join(home, ".config/modot/webapp/profiles", wa.Profile)
		args = append(args, "--user-data-dir="+profileDir)
	}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		args = append(args, "--enable-features=UseOzonePlatform", "--ozone-platform=wayland")
	}

	args = append(args, wa.ExtraFlags...)

	cmd := execdriver.MustRun(ctx, engine, args...)
	executil.InheritContextWriters(ctx, cmd)
	return cmd.Run()
}
