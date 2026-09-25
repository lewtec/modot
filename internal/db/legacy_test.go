package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	lewtest "github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenURLImportsLegacyHistoryOnce(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	t.Setenv("HOME", home)

	legacyPath := filepath.Join(home, ".local", "share", "workspaced", "workspaced.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o755))
	legacy, err := OpenURL(ctx, legacyPath)
	require.NoError(t, err)
	require.NoError(t, legacy.RecordHistory(ctx, types.HistoryEvent{
		Command:   "echo from-old",
		Cwd:       "/tmp",
		Timestamp: 10,
		ExitCode:  0,
		Duration:  1,
	}))
	require.NoError(t, legacy.Close())

	dest := (Arg{}).ArgDefault()
	opened, err := OpenURL(ctx, dest)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, opened)
	empty, err := opened.SearchHistory(ctx, "", 10)
	require.NoError(t, err)
	require.Empty(t, empty)
	require.NoError(t, opened.ImportWorkspacedHistory(ctx))
	got, err := opened.SearchHistory(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "echo from-old", got[0].Command)

	later, err := OpenURL(ctx, legacyPath)
	require.NoError(t, err)
	require.NoError(t, later.RecordHistory(ctx, types.HistoryEvent{
		Command: "echo later", Cwd: "/tmp", Timestamp: 11,
	}))
	require.NoError(t, later.Close())

	again, err := OpenURL(ctx, dest)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, again)
	require.NoError(t, again.ImportWorkspacedHistory(ctx))
	got, err = again.SearchHistory(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestOpenURLImportsLegacyHistoryAsIOTask(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	t.Setenv("HOME", home)

	legacyPath := filepath.Join(home, ".local", "share", "workspaced", "workspaced.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o755))
	legacy, err := OpenURL(ctx, legacyPath)
	require.NoError(t, err)
	require.NoError(t, legacy.RecordHistory(ctx, types.HistoryEvent{
		Command: "echo streamed", Cwd: "/tmp", Timestamp: 3,
	}))
	require.NoError(t, legacy.Close())

	sess, ctx := taskgroup.New(ctx, taskgroup.Limits{IO: 1})
	t.Cleanup(func() { sess.Cancel(context.Canceled) })

	opened, err := OpenURL(ctx, (Arg{}).ArgDefault())
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, opened)
	require.NoError(t, opened.ImportWorkspacedHistory(ctx))
	got, err := opened.SearchHistory(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "echo streamed", got[0].Command)
	require.NoError(t, sess.Wait())
}

func TestOpenURLSkipsLegacyWhenHistoryExists(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	home := t.TempDir()
	t.Setenv("HOME", home)

	dest := (Arg{}).ArgDefault()
	require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0o755))
	first, err := OpenURL(ctx, dest)
	require.NoError(t, err)
	require.NoError(t, first.RecordHistory(ctx, types.HistoryEvent{
		Command: "echo kept", Cwd: "/tmp", Timestamp: 1,
	}))
	require.NoError(t, first.Close())

	legacyPath := filepath.Join(home, ".local", "share", "workspaced", "workspaced.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o755))
	legacy, err := OpenURL(ctx, legacyPath)
	require.NoError(t, err)
	require.NoError(t, legacy.RecordHistory(ctx, types.HistoryEvent{
		Command: "echo old", Cwd: "/tmp", Timestamp: 2,
	}))
	require.NoError(t, legacy.Close())

	opened, err := OpenURL(ctx, dest)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, opened)
	require.NoError(t, opened.ImportWorkspacedHistory(ctx))
	got, err := opened.SearchHistory(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "echo kept", got[0].Command)
}
