package system

import (
	"context"
	"fmt"
	"github.com/lucasew/workspaced/internal/cmdarg"
	"github.com/lucasew/workspaced/internal/cmdctx"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/deployer"
	"github.com/lucasew/workspaced/internal/dotfiles"
	"github.com/lucasew/workspaced/internal/modfile"
	_ "github.com/lucasew/workspaced/internal/modfile/sourceprovider/prelude"
	"github.com/lucasew/workspaced/internal/nix"
	"github.com/lucasew/workspaced/internal/source"
	"github.com/lucasew/workspaced/internal/tool"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
	"github.com/lucasew/workspaced/pkg/logging"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
)

func RunApply(ctx context.Context, action, prefix string) error {
	logger := logging.GetLogger(ctx)
	dryRun := cmdctx.IsDryRun(ctx)
	if prefix == "" {
		prefix = "/"
	}

	dotfilesRoot, err := envdriver.GetDotfilesRoot(ctx)
	if err != nil {
		return fmt.Errorf("get dotfiles root: %w", err)
	}
	cfg, err := configcue.LoadSystem(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ws := modfile.NewWorkspace(dotfilesRoot)
	if _, err := tool.RefreshWorkspaceLocks(ctx, ws, cfg); err != nil {
		return fmt.Errorf("refresh workspace lockfile: %w", err)
	}
	if err := applySystemFiles(ctx, prefix, cfg, ws.ModulesBaseDir(), dryRun); err != nil {
		return err
	}

	if !envdriver.IsNixOS(ctx) || prefix != "/" {
		logger.Info("skipping nixos rebuild", "prefix", prefix)
		return nil
	}

	logger.Info("running NixOS rebuild", "action", action)
	if dryRun {
		logger.Info("dry-run: skipping nixos-rebuild")
		return nil
	}

	flake := ""
	hostname, err := envdriver.GetHostname(ctx)
	if err != nil {
		return fmt.Errorf("hostname: %w", err)
	}
	if hostname == "riverwood" {
		logger.Info("performing remote build for riverwood")
		ref := fmt.Sprintf(".#nixosConfigurations.%s.config.system.build.toplevel", hostname)
		nixResult, err := nix.RemoteBuild(ctx, ref, "whiterun", true)
		if err != nil {
			return fmt.Errorf("remote build failed: %w", err)
		}
		flake = nixResult
	}

	return nix.Rebuild(ctx, action, flake)
}

func applySystemFiles(ctx context.Context, prefix string, cfg *configcue.Config, modulesDir string, dryRun bool) error {
	logger := logging.GetLogger(ctx)
	builder, err := source.StandardDotfilesOptions{
		ConfigTreeTarget: prefix,
		ModulesDir:       modulesDir,
		ModulesCfg:       cfg,
		SystemPrefix:     prefix,
	}.Builder(cfg)
	if err != nil {
		return err
	}
	tree, err := builder.Tree(ctx)
	if err != nil {
		return err
	}
	statePath := lewpath.New(prefix, "var/lib/workspaced/state.json").String()
	stateStore, err := deployer.NewFileStateStore(statePath, prefix)
	if err != nil {
		return fmt.Errorf("create state store: %w", err)
	}
	manager, err := dotfiles.NewManager(dotfiles.Config{
		Tree:       tree,
		StateStore: stateStore,
	})
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}
	logger.Info("applying system files", "prefix", prefix, "files", len(tree.Files()))
	_, err = manager.Apply(ctx, dotfiles.ApplyOptions{DryRun: dryRun})
	return err
}

type Apply struct {
	action cmd.EnumArg[cmdarg.NixAction] `default:"switch"`
	prefix cmd.StringArg                 `long:"prefix" default:"/" help:"system root for module files"`
}

func (Apply) Description() string {
	return "Apply system files under a root, then rebuild NixOS when the root is /"
}

func (a *Apply) Run(ctx context.Context) error {
	return RunApply(ctx, a.action.Value().String(), a.prefix.Value())
}
