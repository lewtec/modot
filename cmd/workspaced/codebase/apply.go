package codebase

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/internal/cmdarg"
	"github.com/lucasew/workspaced/internal/cmdwire"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/deployer"
	"github.com/lucasew/workspaced/internal/dotfiles"
	"github.com/lucasew/workspaced/internal/modfile"
	_ "github.com/lucasew/workspaced/internal/modfile/sourceprovider/prelude"
	"github.com/lucasew/workspaced/internal/source"
	"github.com/lucasew/workspaced/internal/tool"

	"github.com/lewtec/lewkit/x/cmd"
)

type Apply struct {
	ShowNoop cmd.Flag `long:"show-noop" help:"Also show files that would not change"`
	Prefix   Prefix   `long:"prefix" ctx:"prefix" help:"directory that receives codebase files"`
}

func (Apply) Description() string {
	return "Apply modules + templates to the repo root"
}

func (c *Apply) Run(ctx context.Context) error {
	return cmdwire.RunAfterWait(ctx, false, c.ShowNoop.Value(), Schedule)
}

// Schedule wires codebase plan/apply.
// --prefix is the workspace root. Its default is the directory from Prefix.ArgDefault.
func Schedule(ctx context.Context, dryRun, showNoop bool) func() error {
	taskName := "codebase:apply"
	updateMsg := "applying to repo root"
	if dryRun {
		taskName = "codebase:plan"
		updateMsg = "planning changes to repo root"
	}

	logCtx := ctx
	var finalResult *dotfiles.ApplyResult

	taskgroup.Go(ctx, taskName, taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update(updateMsg)
		// Nested plan/apply Maps own aggregate bars; no Unit shell here.

		// Locking uses the same mechanism as home apply:
		// LoadForWorkspace, then RefreshWorkspaceLocks (not force=true mod lock).
		root := cmdarg.PrefixPath(ctx)
		cfg, err := configcue.LoadForWorkspace(ctx, root)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ws := modfile.NewWorkspace(root)
		if _, err := tool.RefreshWorkspaceLocks(ctx, ws, cfg); err != nil {
			return fmt.Errorf("refresh workspace lockfile: %w", err)
		}

		configDir := path.New(root, ".workspaced", "config").String()
		modulesDir := path.New(root, "modules").String()
		stdOpts := source.StandardDotfilesOptions{
			ConfigTreeTarget: root,
			ModulesDir:       modulesDir,
			ModulesCfg:       cfg,
		}
		if _, err := os.Stat(configDir); err == nil {
			stdOpts.ConfigTreeDir = configDir
		}

		b, err := stdOpts.Builder(cfg)
		if err != nil {
			return err
		}
		tree, err := b.Tree(ctx)
		if err != nil {
			return err
		}

		// Repo-local state. Never use the global ~/.config/workspaced state.
		statePath := stateFile(root)
		stateStore, err := deployer.NewFileStateStore(statePath, root)
		if err != nil {
			return fmt.Errorf("create state store: %w", err)
		}

		mgr, err := dotfiles.NewManager(dotfiles.Config{
			Tree:       tree,
			StateStore: stateStore,
			Ignore:     deployer.GitignoreUntracked(root),
		})
		if err != nil {
			return fmt.Errorf("create manager: %w", err)
		}

		result, err := mgr.Apply(ctx, dotfiles.ApplyOptions{
			DryRun: dryRun,
		})
		if err != nil {
			return err
		}

		finalResult = result
		return nil
	})

	return func() error {
		dotfiles.LogApplyResult(logCtx, finalResult, dotfiles.LogApplyOptions{
			ShowNoop:        showNoop,
			DryRun:          dryRun,
			NoChangesTarget: "repo root",
		})
		return nil
	}
}
