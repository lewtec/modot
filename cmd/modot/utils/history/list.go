package history

import (
	"context"
	"os"
	"slices"
	"time"

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
	var lines []historyLine
	for row, err := range database.ListHistory(ctx, int(l.Limit.Value())) {
		if err != nil {
			return err
		}
		lines = append(lines, historyLine{
			Time:    time.Unix(row.Event.Timestamp, 0),
			Count:   row.Count,
			Cwd:     shortenCwd(row.Event.Cwd, home),
			Command: row.Event.Command,
		})
	}
	return cmd.Rows(ctx, os.Stdout, slices.Values(lines), historyLayout)
}
