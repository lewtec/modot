package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/logging"
)

// importLegacyHistory copies command history from the pre-rename sqlite file
// into the default modot database when that database has no history rows yet.
// Ctrl+R reads only modot.db; the old file is otherwise invisible to search.
// The copy is one transaction and one sqlc insert.
func importLegacyHistory(ctx context.Context, d *DB) error {
	if d == nil || d.conn == nil {
		return nil
	}
	dest := (Arg{}).ArgDefault()
	if dest == "" || d.conn.URL() != dest {
		return nil
	}
	got, err := d.Queries.GetHistory(ctx, 1)
	if err != nil || len(got) > 0 {
		return err
	}
	legacy := filepath.Join(filepath.Dir(filepath.Dir(dest)), "workspaced", "workspaced.db")
	if legacy == dest {
		return nil
	}
	if _, err := os.Stat(legacy); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if taskgroup.FromContext(ctx) == nil {
		return d.streamLegacyHistory(ctx, nil, legacy)
	}
	errc := make(chan error, 1)
	taskgroup.Go(ctx, "history:import", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
		err := d.streamLegacyHistory(ctx, s, legacy)
		errc <- err
		return err
	})
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (d *DB) streamLegacyHistory(ctx context.Context, s *taskgroup.Status, legacy string) error {
	dest := d.conn.URL()
	raw, err := sql.Open("sqlite", dest)
	if err != nil {
		return err
	}
	raw.SetMaxOpenConns(1)
	defer logging.Close(ctx, raw, "path", dest)

	if _, err := raw.ExecContext(ctx, `ATTACH DATABASE ? AS srcdb`, legacy); err != nil {
		return err
	}
	defer logging.RunCleanup(ctx, "detach legacy history", func() error {
		_, err := raw.ExecContext(ctx, `DETACH DATABASE srcdb`)
		return err
	})

	tx, err := raw.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			logging.ReportError(ctx, err, "op", "rollback legacy history")
		}
	}()

	var tables int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM srcdb.sqlite_master WHERE type = 'table' AND name = 'history'`).Scan(&tables)
	if err != nil || tables == 0 {
		return err
	}
	// history_src is the sqlc name for the attached table. The view lives only in this transaction.
	if _, err := tx.ExecContext(ctx, `CREATE TEMP VIEW history_src AS SELECT command, cwd, timestamp, exit_code, duration_ms FROM srcdb.history`); err != nil {
		return err
	}

	q := New(dest)(tx)
	total, err := q.CountHistorySrc(ctx)
	if err != nil || total == 0 {
		return err
	}
	existing, err := q.GetHistory(ctx, 1)
	if err != nil || len(existing) > 0 {
		return err
	}
	s.Update("importing history")
	s.Progress(0, total)
	if err := q.CopyAttachedHistory(ctx); err != nil {
		return err
	}
	s.Progress(total, total)
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
