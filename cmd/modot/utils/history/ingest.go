package history

import (
	"context"
	"log/slog"

	"github.com/lewtec/modot/internal/db"
	"github.com/lewtec/modot/internal/types"
)

type Ingest struct {
	Bash       *Bash
	Atuin      *Atuin
	Workspaced *Workspaced
}

func (Ingest) Description() string { return "Ingest history from other sources" }

type Bash struct{}

func (Bash) Description() string { return "Ingest bash history" }

func (Bash) Run(ctx context.Context) error {
	return ingestEvents(ctx, ingestBash)
}

type Atuin struct{}

func (Atuin) Description() string { return "Ingest atuin history" }

func (Atuin) Run(ctx context.Context) error {
	return ingestEvents(ctx, ingestAtuin)
}

type Workspaced struct{}

func (Workspaced) Description() string { return "Ingest history from the pre-rename database" }

func (Workspaced) Run(ctx context.Context) error {
	database, err := db.OpenFromCtx(ctx)
	if err == nil {
		err = database.ImportWorkspacedHistory(ctx)
	}
	return err
}

func ingestEvents(ctx context.Context, load func(context.Context) ([]types.HistoryEvent, error)) error {
	database, err := db.OpenFromCtx(ctx)
	if err != nil {
		return err
	}
	events, err := load(ctx)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		slog.Info("No events to ingest")
		return nil
	}
	slog.Info("Ingesting events...", "amount", len(events))
	return database.BatchRecordHistory(ctx, events)
}
