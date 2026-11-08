package source

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/modot/internal/configcue"
	envdriver "github.com/lewtec/modot/internal/driver/env"

	"github.com/pbnjay/memory"
)

// buildModuleContext returns the data a module template sees under `.module`:
// the module's options from the shared tree, falling back to the whole module
// entry (enable/input/path/from) when the module declares no options.
func buildModuleContext(cfg *configcue.Config, moduleName string) map[string]any {
	if moduleName == "" || cfg == nil {
		return map[string]any{}
	}
	if raw, err := cfg.Lookup("options." + moduleName); err == nil {
		if mapped, ok := raw.(map[string]any); ok && len(mapped) > 0 {
			return mapped
		}
	}
	if raw, err := cfg.Lookup("modules." + moduleName); err == nil {
		if mapped, ok := raw.(map[string]any); ok {
			return mapped
		}
	}
	return map[string]any{}
}

func buildTemplateData(ctx context.Context, cfg *configcue.Config, f File) (map[string]any, error) {
	root := map[string]any{}
	if cfg != nil {
		root = cfg.Raw()
	}

	module := buildModuleContext(cfg, moduleNameOf(f))

	home, err := envdriver.GetHomeDir(ctx)
	if err != nil {
		home, err = envdriver.ResolveHomeDir()
		if err != nil {
			return nil, fmt.Errorf("user home dir: %w", err)
		}
	}
	userDataDir, err := envdriver.GetUserDataDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("user data dir: %w", err)
	}
	hostname, err := envdriver.GetHostname(ctx)
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}

	runtimeData := map[string]any{
		"module_name":   moduleNameOf(f),
		"home":          home,
		"user_data_dir": userDataDir,
		"is_phone":      envdriver.IsPhone(ctx),
		"hostname":      hostname,
		"goos":          runtime.GOOS,
		"goarch":        runtime.GOARCH,
		"memory":        memory.TotalMemory(),
	}
	if dotfilesRoot, err := envdriver.GetDotfilesRoot(ctx); err == nil && dotfilesRoot != "" {
		runtimeData["dotfiles_root"] = dotfilesRoot
	}
	if cfg != nil {
		if mode := cfg.RuntimeMode(); mode != "" {
			runtimeData["mode"] = mode
		}
	}

	out := map[string]any{
		"root":    root,
		"module":  module,
		"runtime": runtimeData,
	}

	return out, nil
}
