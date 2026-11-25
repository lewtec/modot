package rsync

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/executil"
	"github.com/lewtec/modot/internal/logging"
)

// BindStreams returns the writer for rsync stdout+stderr. Chatter stays on
// the session live row; extraOut is an optional transcript (backup notification).
// Context stderr (daemon packet streams) wins over a new live writer.
// Call the returned func after cmd.Run (safe to defer).
func BindStreams(ctx context.Context, extraOut io.Writer) (io.Writer, func()) {
	base := executil.Stderr(ctx)
	var closer io.Closer
	if base == nil {
		lw := taskgroup.LineWriterFrom(ctx)
		base = lw
		closer = lw
	}
	out := base
	if extraOut != nil {
		out = io.MultiWriter(base, extraOut)
	}
	return out, func() {
		if closer != nil {
			logging.Close(ctx, closer)
		}
	}
}

// Sync performs an rsync transfer using the selected driver.
// See Driver.Sync for semantics and taskgroup integration.
func Sync(ctx context.Context, src, dst string, opts Options) error {
	return driver.With(ctx, func(d Driver) error { return d.Sync(ctx, src, dst, opts) })
}

// RunWithTaskGroup is the helper implementations call from their Sync method.
// It ensures that when a taskgroup.Session lives in ctx, the actual transfer
// is executed as a first-class child task ("rsync:src→dst") in the IO pool.
// The perform func receives the child's *Status (for Update/Progress) and
// the caller's opts.Output writer (if any) for transcript forwarding.
func RunWithTaskGroup(
	ctx context.Context,
	src, dst string,
	opts Options,
	perform func(ctx context.Context, st *taskgroup.Status, extraOut io.Writer) error,
) error {
	logger := logging.GetLogger(ctx)
	if taskgroup.FromContext(ctx) == nil {
		// Direct execution (no task tracking). Forward to extra output if provided.
		return perform(ctx, nil, opts.Output)
	}

	name := fmt.Sprintf("rsync:%s", shortName(src, dst))
	errCh := make(chan error, 1)

	taskgroup.Go(ctx, name, taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
		st.Progress(0, -1)
		st.Update("starting")

		err := perform(ctx, st, opts.Output)
		if err != nil {
			st.Update(fmt.Sprintf("error: %v", err))
			logger.Error("rsync task failed", "name", name, "error", err)
		} else {
			st.Progress(1, 1)
			st.Update("done")
			logger.Debug("rsync task completed", "name", name)
		}
		errCh <- err
		return err
	})

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func shortName(src, dst string) string {
	s := lastSegment(src)
	d := lastSegment(dst)
	if s == "" {
		s = src
	}
	if d == "" {
		d = dst
	}
	if len(s) > 30 {
		s = s[:27] + "..."
	}
	if len(d) > 30 {
		d = d[:27] + "..."
	}
	return s + "→" + d
}

func lastSegment(p string) string {
	p = strings.TrimSuffix(p, "/")
	if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
		return p[idx+1:]
	}
	// Handle rsync remote forms like user@host:dir or rsync://host/module/path
	if idx := strings.LastIndex(p, ":"); idx >= 0 && idx < len(p)-1 {
		return p[idx+1:]
	}
	return p
}
