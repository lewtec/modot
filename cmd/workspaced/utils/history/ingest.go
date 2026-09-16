package history

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/workspaced/internal/db"
	"github.com/lucasew/workspaced/internal/types"
	"github.com/lucasew/workspaced/pkg/logging"
)

var ErrUnknownSource = errors.New("unknown source")

type Ingest struct {
	source cmd.StringArg
}

func (Ingest) Description() string { return "Ingest history from other sources (bash, atuin)" }

func (i *Ingest) Run(ctx context.Context) error {
	source := i.source.Value()
	database, err := db.OpenFromCtx(ctx)
	if err != nil {
		return err
	}

	var events []types.HistoryEvent
	switch source {
	case "bash":
		events, err = ingestBash(ctx)
	case "atuin":
		events, err = ingestAtuin(ctx)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownSource, source)
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
