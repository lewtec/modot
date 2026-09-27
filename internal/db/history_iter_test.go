package db

import (
	"path/filepath"
	"testing"

	lewtest "github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/types"
	"github.com/stretchr/testify/require"
)

func TestIterHistoryPrefersCwdAndStops(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	path := filepath.Join(t.TempDir(), "modot.db")
	d, err := OpenURL(ctx, path)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, d)

	rows := []types.HistoryEvent{
		{Command: "ls", Cwd: "/other", Timestamp: 30},
		{Command: "git status", Cwd: "/proj", Timestamp: 20},
		{Command: "pwd", Cwd: "/other", Timestamp: 10},
	}
	require.NoError(t, d.BatchRecordHistory(ctx, rows))

	var got []types.HistoryEvent
	for event, err := range d.IterHistory(ctx, "/proj", 10) {
		require.NoError(t, err)
		got = append(got, event)
		if len(got) == 1 {
			break
		}
	}
	require.Equal(t, []types.HistoryEvent{{Command: "git status", Cwd: "/proj", Timestamp: 20}}, got)

	got = nil
	for event, err := range d.IterHistory(ctx, "/proj", 10) {
		require.NoError(t, err)
		got = append(got, event)
	}
	require.Equal(t, []types.HistoryEvent{
		{Command: "git status", Cwd: "/proj", Timestamp: 20},
		{Command: "ls", Cwd: "/other", Timestamp: 30},
		{Command: "pwd", Cwd: "/other", Timestamp: 10},
	}, got)
}

func TestListHistoryLimitIsNewest(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	path := filepath.Join(t.TempDir(), "modot.db")
	d, err := OpenURL(ctx, path)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, d)

	require.NoError(t, d.BatchRecordHistory(ctx, []types.HistoryEvent{
		{Command: "oldest", Cwd: "/proj", Timestamp: 10},
		{Command: "older", Cwd: "/proj", Timestamp: 20},
		{Command: "newer", Cwd: "/proj", Timestamp: 30},
		{Command: "newest", Cwd: "/other", Timestamp: 40},
	}))

	var got []types.HistoryEvent
	for row, err := range d.ListHistory(ctx, 2) {
		require.NoError(t, err)
		got = append(got, row)
	}
	require.Equal(t, []types.HistoryEvent{
		{Command: "newest", Cwd: "/other", Timestamp: 40},
		{Command: "newer", Cwd: "/proj", Timestamp: 30},
	}, got)
}
