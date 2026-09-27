package sqlite

import (
	"context"
	"iter"
)

const iterHistory = `
SELECT id, command, cwd, timestamp, exit_code, duration_ms FROM history
ORDER BY CASE WHEN cwd = ? THEN 0 ELSE 1 END, timestamp DESC
LIMIT ?
`

const listHistory = `
SELECT command, cwd, timestamp, exit_code, duration_ms, n FROM (
  SELECT
    command,
    cwd,
    timestamp,
    exit_code,
    duration_ms,
    COUNT(*) OVER (PARTITION BY command, cwd) AS n,
    ROW_NUMBER() OVER (PARTITION BY command, cwd ORDER BY timestamp DESC, id DESC) AS rn
  FROM (
    SELECT id, command, cwd, timestamp, exit_code, duration_ms
    FROM history
    ORDER BY timestamp DESC, id DESC
    LIMIT ?
  )
)
WHERE rn = 1
ORDER BY timestamp ASC
`

// Listed is one command in one directory. Count is how many of the newest runs it covers.
type Listed struct {
	History
	Count int64
}

// ListHistory returns one row per command and directory inside the newest limit runs.
// The cursor is chronological: the newest row is last.
func (q *Queries) ListHistory(ctx context.Context, limit int64) iter.Seq2[Listed, error] {
	return func(yield func(Listed, error) bool) {
		rows, err := q.db.QueryContext(ctx, listHistory, limit)
		if err != nil {
			yield(Listed{}, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var row Listed
			if err := rows.Scan(&row.Command, &row.Cwd, &row.Timestamp, &row.ExitCode, &row.DurationMs, &row.Count); err != nil {
				yield(Listed{}, err)
				return
			}
			if !yield(row, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(Listed{}, err)
		}
	}
}

// IterHistory streams history newest-first. Rows whose cwd equals preferCwd come first.
// An empty preferCwd keeps timestamp order. The LIMIT is applied by sqlite.
func (q *Queries) IterHistory(ctx context.Context, preferCwd string, limit int64) iter.Seq2[History, error] {
	return func(yield func(History, error) bool) {
		rows, err := q.db.QueryContext(ctx, iterHistory, preferCwd, limit)
		if err != nil {
			yield(History{}, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var row History
			if err := rows.Scan(&row.ID, &row.Command, &row.Cwd, &row.Timestamp, &row.ExitCode, &row.DurationMs); err != nil {
				yield(History{}, err)
				return
			}
			if !yield(row, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(History{}, err)
		}
	}
}
