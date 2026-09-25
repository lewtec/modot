package history

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/db"
	"github.com/lewtec/modot/internal/types"
	"github.com/lewtec/modot/internal/logging"
)

type Ingest struct {
	source cmd.EnumArg[cmdarg.HistorySource]
}

func (Ingest) Description() string { return "Ingest history from other sources (bash, atuin)" }

func (i *Ingest) Run(ctx context.Context) error {
	database, err := db.OpenFromCtx(ctx)
	if err != nil {
		return err
	}

	var events []types.HistoryEvent
	switch i.source.Value() {
	case cmdarg.HistoryAtuin:
		events, err = ingestAtuin(ctx)
	default:
		events, err = ingestBash(ctx)
	}

	if err != nil {
		return err
	}

	if len(events) == 0 {
		logger := logging.GetLogger(ctx)
		logger.Info("No events to ingest")
		return nil
	}

	logger := logging.GetLogger(ctx)
	logger.Info("Ingesting events...", "amount", len(events))
	return database.BatchRecordHistory(ctx, events)
}
