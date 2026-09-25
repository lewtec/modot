package native

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/driver"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	rsyncdriver "github.com/lewtec/modot/internal/driver/rsync"
	"github.com/lewtec/modot/internal/executil"
	"github.com/lewtec/modot/internal/logging"
)

// ErrBinaryNotAvailable is returned when execRsync runs without an rsync binary on PATH.
var ErrBinaryNotAvailable = errors.New("rsync binary not available")

func init() {
	driver.Register[rsyncdriver.Driver](&Factory{})
}

type Factory struct{}

func (f *Factory) ID() string   { return "rsync_native" }
func (f *Factory) Name() string { return "Native rsync" }

func (f *Factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireBinary(ctx, "rsync")
}

func (f *Factory) New(ctx context.Context) (rsyncdriver.Driver, error) {
	return &Driver{}, nil
}

type Driver struct{}

func (d *Driver) Sync(ctx context.Context, src, dst string, opts rsyncdriver.Options) error {
	logger := logging.GetLogger(ctx)
	return rsyncdriver.SyncWith(ctx, src, dst, opts, []string{"-avP"},
		func(ctx context.Context, args []string, st *taskgroup.Status, extraOut io.Writer) error {
			return d.execRsync(ctx, args, st, extraOut, logger)
		})
}

func (d *Driver) execRsync(ctx context.Context, args []string, st *taskgroup.Status, extraOut io.Writer, logger *slog.Logger) error {
	if !execdriver.IsBinaryAvailable(ctx, "rsync") {
		return ErrBinaryNotAvailable
	}

	cmd := execdriver.MustRun(ctx, "rsync", args...)
	base := cmd.Stderr
	out := io.Writer(base)
	if extraOut != nil {
		out = io.MultiWriter(base, extraOut)
	}
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	if extraOut != nil && executil.Stderr(ctx) == nil {
		if c, ok := base.(io.Closer); ok {
			logging.Close(ctx, c)
		}
	}
	return err
}
