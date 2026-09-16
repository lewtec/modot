package main

//go:generate go run github.com/lucasew/workspaced/internal/devtools/autoregistry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"

	cueerrors "cuelang.org/go/cue/errors"
	pkg_daemon "github.com/lucasew/workspaced/cmd/workspaced/daemon"
	"github.com/lucasew/workspaced/internal/afterwait"
	"github.com/lucasew/workspaced/internal/cmdctx"
	"github.com/lucasew/workspaced/internal/configcue"
	_ "github.com/lucasew/workspaced/internal/tool/prelude"
	"github.com/lucasew/workspaced/internal/version"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
	_ "github.com/lucasew/workspaced/pkg/driver/prelude"
	"github.com/lucasew/workspaced/pkg/logging"
	_ "github.com/lucasew/workspaced/pkg/palette/prelude"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

var (
	processLogger *slog.Logger
	processLogOut *swapWriter
)

func main() {
	level := &slog.LevelVar{}
	processLogOut = newSwapWriter(os.Stderr)
	processLogger = slog.New(logging.NewPlainHandler(processLogOut, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(processLogger)
	rootCtx := logging.NewRootContext(processLogger)

	if os.Getenv("REBUILD_TEST") != "" {
		exe, err := os.Executable()
		if err != nil {
			panic(err)
		}
		h := sha256.New()
		f, err := os.Open(exe)
		if err != nil {
			panic(err)
		}
		defer logging.Close(rootCtx, f, "path", exe)
		if _, err = io.Copy(h, f); err != nil {
			panic(err)
		}
		logging.GetLogger(rootCtx).Info("build time", "t", h.Sum(nil))
	}
	if _, err := configcue.LoadHome(rootCtx); err != nil {
		logging.GetLogger(rootCtx).Debug("failed to load config", "error", err)
	}

	pkg_daemon.ExecuteCLI = executeCLI

	if err := run(rootCtx, level); err != nil {
		logger := logging.GetLogger(rootCtx)
		if details := cueerrors.Details(err, nil); details != "" {
			logger.Error("error", "err", err, "details", "\n"+details)
		} else {
			logger.Error("error", "err", err)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, level *slog.LevelVar) error {
	app, err := cmd.Parse[cmd.App[cli]](os.Args[1:]...)
	if err != nil {
		return err
	}
	level.Set(app.LogLevel())
	if app.WantVersion() {
		_, err := fmt.Fprintln(os.Stdout, version.VersionString())
		return err
	}
	if app.Help() || !selectedHasRun(reflect.ValueOf(&app.Args).Elem()) {
		return app.Run(ctx)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx, session, err := setup(ctx, app)
	if err != nil {
		return err
	}
	runErr := progress.Run(session, ctx, func(ctx context.Context) error {
		if processLogOut != nil {
			processLogOut.Set(session.LogWriter())
		}
		return app.Run(ctx)
	})
	if processLogOut != nil {
		processLogOut.Set(os.Stderr)
	}
	if hookErr := afterwait.Run(ctx); runErr == nil {
		runErr = hookErr
	}
	cancel()
	if runErr != nil {
		logging.GetLogger(ctx).Error("task group error", "err", runErr)
	}
	return runErr
}

func executeCLI(ctx context.Context, args []string) error {
	app, err := cmd.Parse[cmd.App[cli]](args...)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}

func setup(ctx context.Context, app cmd.App[cli]) (context.Context, *taskgroup.Session, error) {
	envdriver.SetupEssentialPaths(ctx)
	ctx = cmdctx.WithDryRun(ctx, app.Args.DryRun.Value())
	ctx = afterwait.With(ctx)
	armedNoCache := app.Args.NoCache.Value()
	ctx = cmdctx.WithNoCache(ctx, armedNoCache)
	if armedNoCache {
		logging.GetLogger(ctx).Info("no-cache enabled (flag or WORKSPACED_NO_CACHE)")
	}

	base := taskgroup.DefaultLimits()
	if homeCfg, err := configcue.LoadHome(ctx); err == nil {
		base = homeCfg.ConcurrencyLimits()
	}
	session, ctx := app.Args.Enter(ctx, base)
	return ctx, session, nil
}
