package gokrazy

import (
	"context"
	"io"
	"log/slog"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/driver"
	rsyncdriver "github.com/lewtec/modot/internal/driver/rsync"
	"github.com/lewtec/modot/internal/logging"

	gokrsync "github.com/gokrazy/rsync/rsynccmd"
)

func init() {
	driver.Register[rsyncdriver.Driver](&Factory{})
}

type Factory struct{}

func (f *Factory) ID() string   { return "rsync_gokrazy" }
func (f *Factory) Name() string { return "gokrazy/rsync (pure Go)" }

func (f *Factory) CheckCompatibility(ctx context.Context) error {
	// Pure Go implementation, always available.
	return nil
}

func (f *Factory) New(ctx context.Context) (rsyncdriver.Driver, error) {
	return &Driver{}, nil
}

type Driver struct{}

func (d *Driver) Sync(ctx context.Context, src, dst string, opts rsyncdriver.Options) error {
	logger := logging.GetLogger(ctx)
	// gokrazy/rsync: avoid -P/--partial (not implemented); use long --progress.
	return rsyncdriver.SyncWith(ctx, src, dst, opts, []string{"-av", "--progress"},
		func(ctx context.Context, args []string, st *taskgroup.Status, extraOut io.Writer) error {
			return d.runRsyncCmd(ctx, args, st, extraOut, logger)
		})
}

func (d *Driver) runRsyncCmd(ctx context.Context, args []string, st *taskgroup.Status, extraOut io.Writer, logger *slog.Logger) error {
	// rsynccmd gives us a drop-in replacement for spawning rsync.
	cmd := gokrsync.Command("rsync", args...)
	out, done := rsyncdriver.BindStreams(ctx, extraOut)
	defer done()
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.DontRestrict = true

	_, err := cmd.Run(ctx)
	return err
}
