package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/lewtec/modot/internal/logging"
)

// importLegacyHistory copies command history from the pre-rename sqlite file
// into the default modot database when that database has no history rows yet.
// Ctrl+R reads only modot.db; the old file is otherwise invisible to search.
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
	rows, err := src.QueryContext(ctx, `SELECT command, cwd, timestamp, exit_code, duration_ms FROM history`)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, rows)

	var events []RecordHistoryParams
	for rows.Next() {
		var ev RecordHistoryParams
		if err := rows.Scan(&ev.Command, &ev.Cwd, &ev.Timestamp, &ev.ExitCode, &ev.DurationMs); err != nil {
			return err
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	return d.conn.Tx(ctx, func(q Queries) error {
		again, err := q.GetHistory(ctx, 1)
		if err != nil || len(again) > 0 {
			return err
		}
		for _, ev := range events {
			if err := q.RecordHistory(ctx, ev); err != nil {
				return err
			}
		}
		return nil
	})
}
