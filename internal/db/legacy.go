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
// The copy is one INSERT SELECT, so the rows are not buffered in the process.
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
	return streamLegacyHistory(ctx, dest, legacy)
}

func streamLegacyHistory(ctx context.Context, dest, legacy string) error {
	conn, err := sql.Open("sqlite", dest)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, conn, "path", dest)

	if _, err := conn.ExecContext(ctx, `ATTACH DATABASE ? AS legacy`, legacy); err != nil {
		return err
	}
	defer logging.RunCleanup(ctx, "detach legacy history", func() error {
		_, err := conn.ExecContext(ctx, `DETACH DATABASE legacy`)
		return err
	})

	var table string
	err = conn.QueryRowContext(ctx, `SELECT name FROM legacy.sqlite_master WHERE type = 'table' AND name = 'history'`).Scan(&table)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if _, err := conn.ExecContext(ctx, `ROLLBACK`); err != nil {
			logging.ReportError(ctx, err, "op", "rollback legacy history")
		}
	}()

	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM main.history`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO main.history (command, cwd, timestamp, exit_code, duration_ms)
		SELECT command, cwd, timestamp, exit_code, duration_ms FROM legacy.history`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return err
	}
	committed = true
	return nil
}
