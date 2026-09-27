package history

import (
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/text/table"
	"github.com/lewtec/modot/internal/types"
	"github.com/stretchr/testify/require"
)

func TestShortenCwd(t *testing.T) {
	home := "/home/lucasew"
	require.Equal(t, "~", shortenCwd(home, home))
	require.Equal(t, "~/WORKSPACE/LEWTEC/lewkit", shortenCwd(home+"/WORKSPACE/LEWTEC/lewkit", home))
	require.Equal(t, "/tmp", shortenCwd("/tmp", home))
	require.Equal(t, "/home/lucasew", shortenCwd(home, ""))
}

func TestFinderLineOmitsDirectory(t *testing.T) {
	line := finderLine(types.HistoryEvent{
		Command: "git status",
		Cwd:     "/home/lucasew/WORKSPACE/LEWTEC/lewkit",
	})
	require.Equal(t, "git status", line)
}

func TestPreviewFits(t *testing.T) {
	require.False(t, previewFits(79))
	require.True(t, previewFits(80))
}

func TestCollapseByCommandCwd(t *testing.T) {
	events := []types.HistoryEvent{
		{Command: "sd g s", Cwd: "/proj", Timestamp: 40, ExitCode: 0},
		{Command: "sd g s", Cwd: "/other", Timestamp: 30, ExitCode: 1},
		{Command: "sd g s", Cwd: "/proj", Timestamp: 20, ExitCode: 2},
		{Command: "ls", Cwd: "/proj", Timestamp: 10, ExitCode: 0},
	}
	rows, err := collapseHistory(iter.Seq2[types.HistoryEvent, error](func(yield func(types.HistoryEvent, error) bool) {
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}))
	require.NoError(t, err)
	require.Equal(t, []collapsedRow{
		{Event: events[0], Count: 2},
		{Event: events[1], Count: 1},
		{Event: events[3], Count: 1},
	}, rows)
}

func TestReverseCollapsedPutsNewestLast(t *testing.T) {
	events := []types.HistoryEvent{
		{Command: "newest", Cwd: "/tmp", Timestamp: 40},
		{Command: "middle", Cwd: "/proj", Timestamp: 30},
		{Command: "middle", Cwd: "/proj", Timestamp: 20},
		{Command: "oldest", Cwd: "/proj", Timestamp: 10},
	}
	rows, err := collapseHistory(iter.Seq2[types.HistoryEvent, error](func(yield func(types.HistoryEvent, error) bool) {
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}))
	require.NoError(t, err)
	slices.Reverse(rows)
	require.Equal(t, []string{"oldest", "middle", "newest"}, []string{rows[0].Event.Command, rows[1].Event.Command, rows[2].Event.Command})
	require.Equal(t, 2, rows[1].Count)
	require.Equal(t, int64(30), rows[1].Event.Timestamp)
}

func TestHistoryTableNewestLast(t *testing.T) {
	rows := []collapsedRow{
		{Event: types.HistoryEvent{Command: "oldest", Cwd: "/home/lucasew/proj", Timestamp: 1_700_000_050}, Count: 1},
		{Event: types.HistoryEvent{Command: "middle", Cwd: "/home/lucasew/proj", Timestamp: 1_700_000_100}, Count: 1},
		{Event: types.HistoryEvent{Command: "newest", Cwd: "/tmp", Timestamp: 1_700_000_200}, Count: 4},
	}
	var buf strings.Builder
	require.NoError(t, table.Write(&buf, table.Table, historyLines(rows, "/home/lucasew"), historyBriefLayout))
	text := buf.String()
	oldest := strings.Index(text, "oldest")
	middle := strings.Index(text, "middle")
	newest := strings.Index(text, "newest")
	require.Less(t, oldest, middle)
	require.Less(t, middle, newest)
	require.NotContains(t, text, "~/proj")
	require.NotContains(t, text, "cwd")

	buf.Reset()
	require.NoError(t, table.Write(&buf, table.Table, historyLines(rows, "/home/lucasew"), historyLayout))
	require.Contains(t, buf.String(), "~/proj")
}
