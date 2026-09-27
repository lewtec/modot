package history

import (
	"context"
	"os"
	"slices"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/db"
)

type List struct {
	cmd.Output `flatten:""`
	Limit      cmd.IntArg[int32] `long:"limit" help:"Limit number of entries" default:"5000"`
}

func (List) Description() string { return "List history entries, newest last" }

func (l *List) Run(ctx context.Context) error {
	database, err := db.OpenFromCtx(ctx)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	rows, err := collapseHistory(database.ListHistory(ctx, int(l.Limit.Value())))
	if err != nil {
		return err
	}
	slices.Reverse(rows)
	layout := historyBriefLayout
	if l.Columns.Value() != "" {
		layout = historyLayout
	}
	return cmd.Rows(ctx, os.Stdout, historyLines(rows, home), layout)
}
