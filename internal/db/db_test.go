package db

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	lewtest "github.com/lewtec/lewkit/x/test"
	"github.com/lucasew/workspaced/internal/types"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenURLMigratesAndQueries(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	path := filepath.Join(t.TempDir(), "workspaced.db")
	d, err := OpenURL(ctx, path)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, d)

	ev := types.HistoryEvent{
		Command:   "workspaced home apply",
		Cwd:       "/tmp",
		Timestamp: 1,
		ExitCode:  0,
		Duration:  10,
	}
	require.NoError(t, d.RecordHistory(ctx, ev))
	got, err := d.SearchHistory(ctx, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, ev.Command, got[0].Command)
}

func TestOpenArgUsesParsedFlag(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	path := filepath.Join(t.TempDir(), "flag.db")
	var a Arg
	require.NoError(t, a.Parse(path))
	d, err := OpenArg(ctx, a)
	require.NoError(t, err)
	lewtest.CloseOnCleanup(t, d)
	require.NotNil(t, d)
}

func TestArgDefaultIsEmpty(t *testing.T) {
	got := cmd.ParseOK[struct {
		DB Arg `long:"database"`
	}](t)
	require.NotNil(t, got.DB.Value())
	assert.Empty(t, got.DB.Value().URL())
}

func TestOpenURLRejectsUnknownScheme(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	_, err := OpenURL(ctx, "postgres://localhost/app")
	require.Error(t, err)
}
