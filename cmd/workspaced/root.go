package main

//go:generate go run github.com/lucasew/workspaced/internal/devtools/autoregistry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"

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
)

var processLogger *slog.Logger

func main() {
	level := &slog.LevelVar{}
	processLogger = slog.New(logging.NewPlainHandler(logging.ProcessWriter(), &slog.HandlerOptions{
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
	ctx = prepare(ctx, app)
	runErr := app.Run(ctx)
	if hookErr := afterwait.Run(ctx); runErr == nil {
		runErr = hookErr
	}
	return runErr
}

func executeCLI(ctx context.Context, args []string) error {
	app, err := cmd.Parse[cmd.App[cli]](args...)
	if err != nil {
		return err
	}
	ctx = afterwait.With(ctx)
	runErr := app.Run(ctx)
	if hookErr := afterwait.Run(ctx); runErr == nil {
		runErr = hookErr
	}
	return runErr
}

func prepare(ctx context.Context, app cmd.App[cli]) context.Context {
	envdriver.SetupEssentialPaths(ctx)
	ctx = cmdctx.WithDryRun(ctx, app.Args.DryRun.Value())
	ctx = afterwait.With(ctx)
	armedNoCache := app.Args.NoCache.Value()
	ctx = cmdctx.WithNoCache(ctx, armedNoCache)
	if armedNoCache {
		logging.GetLogger(ctx).Info("no-cache enabled (flag or WORKSPACED_NO_CACHE)")
	}
	return ctx
}
