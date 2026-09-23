package codebase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	ShowNoop cmd.Flag      `long:"show-noop" help:"Also show files that would not change"`
	Prefix   cmdarg.Prefix `long:"prefix" ctx:"prefix" default:"." help:"directory that receives codebase files"`
}

func (Apply) Description() string {
	return "Apply modules + templates to the repo root"
}

func (c *Apply) Run(ctx context.Context) error {
	return cmdwire.RunAfterWait(ctx, false, c.ShowNoop.Value(), Schedule)
}

// Schedule wires codebase plan/apply.
// The apply target is --prefix. The workspace root is where the cue and modules are read.
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

		// Discover the closest workspaced.cue from current CWD (or fall back
		// to git root). The directory containing the cue is the workspace root
		// for this run: both the apply target and the lockfile location.
		//
		// This is deliberate. "codebase" is the general mechanism for operating
		// on *any* repo/tree that has a workspaced.cue (including sub-projects,
		// skill trees, random checkouts, the dotfiles repo itself, etc.).
		// It must not reach out to the user's personal dotfiles root.
		//
		// Locking uses the same *mechanism* as home apply:
		//   - Load config anchored to the specific workspace root
		//     (LoadForWorkspace, like home uses LoadHome for its root)
		//   - RefreshWorkspaceLocks (non-force path for sources + lazy tools)
		//     instead of the force=true mod lock path.
		// This makes ref/hash filling, skipping of already-locked HEAD inputs,
		// and tool lock enrichment behave consistently.
		cuePath, err := configcue.ResolveWorkspaceCuePath(ctx, "")
		if err != nil {
			return fmt.Errorf("resolve workspaced.cue: %w", err)
		}
		workspaceRoot := ""
		if cuePath != "" {
			workspaceRoot = filepath.Dir(cuePath)
		} else {
			// Fallback to git root (or dotfiles root as last resort)
			ws, err := modfile.DetectWorkspace(ctx, "")
			if err != nil {
				return fmt.Errorf("detect workspace: %w", err)
			}
			workspaceRoot = ws.Root
		}

		cfg, err := configcue.LoadForWorkspace(ctx, workspaceRoot)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ws := modfile.NewWorkspace(workspaceRoot)
		if _, err := tool.RefreshWorkspaceLocks(ctx, ws, cfg); err != nil {
			return fmt.Errorf("refresh workspace lockfile: %w", err)
		}

		configDir := filepath.Join(workspaceRoot, ".workspaced", "config")
		modulesDir := filepath.Join(workspaceRoot, "modules")

		target := cmdarg.PrefixPath(ctx)
		if target == "" {
			return fmt.Errorf("prefix is not set")
		}
		stdOpts := source.StandardDotfilesOptions{
			ConfigTreeTarget: target,
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

		// State lives next to the apply target.
		// Repo-local state for codebase operations. Never use the global
		// ~/.config/workspaced state. Paths on disk are relative to the prefix.
		statePath := filepath.Join(target, ".workspaced", "state.json")
		stateStore, err := deployer.NewFileStateStore(statePath, target)
		if err != nil {
			return fmt.Errorf("create state store: %w", err)
		}

		mgr, err := dotfiles.NewManager(dotfiles.Config{
			Tree:       tree,
			StateStore: stateStore,
			Ignore:     deployer.GitignoreUntracked(workspaceRoot),
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
