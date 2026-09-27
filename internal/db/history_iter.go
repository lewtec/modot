package db

import (
	"context"
	"errors"
	"iter"

	"github.com/lewtec/modot/internal/types"
)

func (w sqliteWrap) iterHistory(ctx context.Context, preferCwd string, limit int64) iter.Seq2[History, error] {
	return func(yield func(History, error) bool) {
		for row, err := range w.q.IterHistory(ctx, preferCwd, limit) {
			if err != nil {
				yield(History{}, err)
				return
			}
			if !yield(History(row), nil) {
				return
			}
		}
	}
}

// IterHistory streams one row at a time. preferCwd sorts that directory ahead of the others.
func (d *DB) IterHistory(ctx context.Context, preferCwd string, limit int) iter.Seq2[types.HistoryEvent, error] {
	return func(yield func(types.HistoryEvent, error) bool) {
		if d == nil || d.conn == nil {
			yield(types.HistoryEvent{}, errors.New("database url not set"))
			return
		}
		src, ok := d.Queries.(interface {
			iterHistory(context.Context, string, int64) iter.Seq2[History, error]
		})
		if !ok {
			yield(types.HistoryEvent{}, errors.New("history cursor is not supported"))
			return
		}
		for row, err := range src.iterHistory(ctx, preferCwd, int64(limit)) {
			if err != nil {
				yield(types.HistoryEvent{}, err)
				return
			}
			if !yield(historyEvent(row), nil) {
				return
			}
		}
	}
}

// ListedHistory is one collapsed history row. The iterator order is chronological, newest last.
type ListedHistory struct {
	Event types.HistoryEvent
	Count int
}

func (w sqliteWrap) listHistory(ctx context.Context, limit int64) iter.Seq2[ListedHistory, error] {
	return func(yield func(ListedHistory, error) bool) {
		for row, err := range w.q.ListHistory(ctx, limit) {
			if err != nil {
				yield(ListedHistory{}, err)
				return
			}
			if !yield(ListedHistory{Event: historyEvent(History(row.History)), Count: int(row.Count)}, nil) {
				return
			}
		}
	}
}

// ListHistory streams collapsed rows oldest-first. The order comes from the query.
func (d *DB) ListHistory(ctx context.Context, limit int) iter.Seq2[ListedHistory, error] {
	return func(yield func(ListedHistory, error) bool) {
		if d == nil || d.conn == nil {
			yield(ListedHistory{}, errors.New("database url not set"))
			return
		}
		src, ok := d.Queries.(interface {
			listHistory(context.Context, int64) iter.Seq2[ListedHistory, error]
		})
		if !ok {
			yield(ListedHistory{}, errors.New("history cursor is not supported"))
			return
		}
		for row, err := range src.listHistory(ctx, int64(limit)) {
			if err != nil {
				yield(ListedHistory{}, err)
				return
			}
			if !yield(row, nil) {
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
