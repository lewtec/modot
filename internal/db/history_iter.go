package db

import (
	"context"
	"errors"
	"iter"

	"github.com/lewtec/modot/internal/types"
)

// IterHistory yields HistoryPreferCwd. preferCwd sorts that directory ahead of the others.
func (d *DB) IterHistory(ctx context.Context, preferCwd string, limit int) iter.Seq2[types.HistoryEvent, error] {
	return func(yield func(types.HistoryEvent, error) bool) {
		if d == nil || d.conn == nil || d.Queries == nil {
			yield(types.HistoryEvent{}, errors.New("database url not set"))
			return
		}
		rows, err := d.Queries.HistoryPreferCwd(ctx, HistoryPreferCwdParams{
			PreferCwd: preferCwd,
			RowLimit:  int64(limit),
		})
		if err != nil {
			yield(types.HistoryEvent{}, err)
			return
		}
		for _, row := range rows {
			if !yield(historyEvent(row), nil) {
				return
			}
		}
	}
}

// ListHistory yields RecentHistory, newest run first. The limit is applied in that order.
func (d *DB) ListHistory(ctx context.Context, limit int) iter.Seq2[types.HistoryEvent, error] {
	return func(yield func(types.HistoryEvent, error) bool) {
		if d == nil || d.conn == nil || d.Queries == nil {
			yield(types.HistoryEvent{}, errors.New("database url not set"))
			return
		}
		rows, err := d.Queries.RecentHistory(ctx, int64(limit))
		if err != nil {
			yield(types.HistoryEvent{}, err)
			return
		}
		for _, row := range rows {
			if !yield(historyEvent(row), nil) {
				return
			}
		}
	}
}

func historyEvent(row History) types.HistoryEvent {
	return types.HistoryEvent{
		Command:   row.Command,
		Cwd:       row.Cwd,
		Timestamp: row.Timestamp,
		ExitCode:  int(row.ExitCode),
		Duration:  row.DurationMs,
	}
}
