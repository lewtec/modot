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
// Rows are inserted through RecordHistory as they are read.
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
	src, err := sql.Open("sqlite", legacy)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, src, "path", legacy)

	var table string
	err = src.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'history'`).Scan(&table)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var total int64
	if err := src.QueryRowContext(ctx, `SELECT COUNT(*) FROM history`).Scan(&total); err != nil {
		return err
	}
	if total == 0 {
		return nil
	}
	s.Update("importing history")
	s.Progress(0, total)

	rows, err := src.QueryContext(ctx, `SELECT command, cwd, timestamp, exit_code, duration_ms FROM history`)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, rows)

	return d.conn.Tx(ctx, func(q Queries) error {
		again, err := q.GetHistory(ctx, 1)
		if err != nil || len(again) > 0 {
			return err
		}
		var n int64
		for rows.Next() {
			var ev RecordHistoryParams
			if err := rows.Scan(&ev.Command, &ev.Cwd, &ev.Timestamp, &ev.ExitCode, &ev.DurationMs); err != nil {
				return err
			}
			if err := q.RecordHistory(ctx, ev); err != nil {
				return err
			}
			n++
			s.Progress(n, total)
		}
		return rows.Err()
	})
}
