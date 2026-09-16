// Package db is the workspaced sqlite store.
//
// Queries and migrations live under sqlite/. go generate runs
// lewkit generate db (sqlc output, FS, Queries, New, DBArg).
package db

//go:generate go tool lewkit generate db .

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	xdb "github.com/lewtec/lewkit/x/db"
	"github.com/lucasew/workspaced/internal/types"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
)

type dbKey struct{}

// WithDB returns a context that carries the given database connection.
func WithDB(ctx context.Context, database *DB) context.Context {
	return context.WithValue(ctx, dbKey{}, database)
}

// FromContext retrieves the database connection from the context.
func FromContext(ctx context.Context) (*DB, bool) {
	database, ok := ctx.Value(dbKey{}).(*DB)
	return database, ok
}

type DB struct {
	conn    *xdb.Conn[Queries]
	Queries Queries
}

// Open parses ArgDefault and opens it. Same path as --database with no flag.
func Open(ctx context.Context) (*DB, error) {
	var a Arg
	if err := a.Parse(a.ArgDefault()); err != nil {
		return nil, err
	}
	return OpenArg(ctx, a)
}

// Arg is --database. ArgDefault is ~/.local/share/workspaced/workspaced.db
// (Termux home rewrite via ResolveHomeDir).
type Arg struct {
	DBArg
}

func (Arg) ArgDefault() string {
	home, err := envdriver.ResolveHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "workspaced", "workspaced.db")
}

// OpenURL opens a sqlite URL (bare path, file:, sqlite:, or :memory:)
// and applies sqlite/migrations.
func OpenURL(ctx context.Context, url string) (*DB, error) {
	var a Arg
	if err := a.Parse(url); err != nil {
		return nil, err
	}
	return OpenArg(ctx, a)
}

// OpenArg opens a parsed Arg. Callers must Close the result.
func OpenArg(ctx context.Context, a Arg) (*DB, error) {
	c := a.Value()
	if c == nil {
		return nil, fmt.Errorf("database url not set")
	}
	if u := c.URL(); u != "" && u != ":memory:" && !strings.Contains(u, "://") && !strings.HasPrefix(u, "file:") {
		if err := os.MkdirAll(filepath.Dir(u), 0o755); err != nil {
			return nil, err
		}
	}
	if err := a.Open(ctx); err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return FromArg(a.DBArg), nil
}

// FromArg wraps an already-Open DBArg. Nil if the arg was never parsed.
func FromArg(a DBArg) *DB {
	conn := a.Value()
	if conn == nil {
		return nil
	}
	return &DB{conn: conn, Queries: conn.Queries()}
}

func (d *DB) Close() error {
	if d == nil || d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

func (d *DB) RecordHistory(ctx context.Context, event types.HistoryEvent) error {
	return d.Queries.RecordHistory(ctx, RecordHistoryParams{
		Command:    event.Command,
		Cwd:        event.Cwd,
		Timestamp:  event.Timestamp,
		ExitCode:   int64(event.ExitCode),
		DurationMs: event.Duration,
	})
}

func (d *DB) BatchRecordHistory(ctx context.Context, events []types.HistoryEvent) error {
	return d.conn.Tx(ctx, func(q Queries) error {
		for _, event := range events {
			if err := q.RecordHistory(ctx, RecordHistoryParams{
				Command:    event.Command,
				Cwd:        event.Cwd,
				Timestamp:  event.Timestamp,
				ExitCode:   int64(event.ExitCode),
				DurationMs: event.Duration,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) SearchHistory(ctx context.Context, query string, limit int) ([]types.HistoryEvent, error) {
	var rows []History
	var err error
	limit64 := int64(limit)
	if query == "" {
		rows, err = d.Queries.GetHistory(ctx, limit64)
	} else {
		rows, err = d.Queries.SearchHistory(ctx, SearchHistoryParams{
			Command: "%" + query + "%",
			Limit:   limit64,
		})
	}
	if err != nil {
		return nil, err
	}
	events := make([]types.HistoryEvent, len(rows))
	for i, row := range rows {
		events[i] = types.HistoryEvent{
			Command:   row.Command,
			Cwd:       row.Cwd,
			Timestamp: row.Timestamp,
			ExitCode:  int(row.ExitCode),
			Duration:  row.DurationMs,
		}
	}
	return events, nil
}
