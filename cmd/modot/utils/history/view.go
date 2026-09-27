package history

import (
	"iter"
	"os"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/text/table"
	"github.com/lewtec/modot/internal/types"
)

func shortenCwd(dir, home string) string {
	if home == "" {
		return dir
	}
	if dir == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(dir, prefix) {
		return "~" + string(os.PathSeparator) + strings.TrimPrefix(dir, prefix)
	}
	return dir
}

func finderLine(event types.HistoryEvent, home string) string {
	return shortenCwd(event.Cwd, home) + "  " + event.Command
}

type collapsedRow struct {
	Event types.HistoryEvent
	Count int
}

func collapseHistory(seq iter.Seq2[types.HistoryEvent, error]) ([]collapsedRow, error) {
	index := map[string]int{}
	var rows []collapsedRow
	for event, err := range seq {
		if err != nil {
			return nil, err
		}
		key := event.Command + "\x00" + event.Cwd
		if i, ok := index[key]; ok {
			rows[i].Count++
			continue
		}
		index[key] = len(rows)
		rows = append(rows, collapsedRow{Event: event, Count: 1})
	}
	return rows, nil
}

type historyLine struct {
	Time    time.Time `json:"time"`
	Count   int       `json:"count"`
	Cwd     string    `json:"cwd"`
	Command string    `json:"command"`
}

type historySpec struct {
	Time    table.Field[time.Time]
	Count   table.Field[int]
	Cwd     table.Field[string]
	Command table.Field[string]
}

var historyLayout = table.Must(table.Make[historyLine](historySpec{
	Time: table.Field[time.Time]{Format: time.DateTime},
}))

func historyLines(rows []collapsedRow, home string) iter.Seq[historyLine] {
	return func(yield func(historyLine) bool) {
		for _, row := range rows {
			line := historyLine{
				Time:    time.Unix(row.Event.Timestamp, 0),
				Count:   row.Count,
				Cwd:     shortenCwd(row.Event.Cwd, home),
				Command: row.Event.Command,
			}
			if !yield(line) {
				return
			}
		}
	}
}
