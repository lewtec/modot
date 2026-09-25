package codebase

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/modfile"
	_ "github.com/lewtec/modot/internal/driver/exec/native"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestPrefixFlagWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	type args struct {
		Prefix Prefix `long:"prefix"`
	}
	got, err := cmd.Parse[args]("--prefix", dir)
	require.NoError(t, err)
	require.Equal(t, dir, got.Prefix.Value())
	want := lewpath.New(".modot", "state.json")
	require.Equal(t, want, got.Prefix.StatePath())
}

func TestPrefixDefaultIsWorkspaceRoot(t *testing.T) {
	t.Parallel()
	type args struct {
		Prefix Prefix `long:"prefix"`
	}
	got, err := cmd.Parse[args]()
	require.NoError(t, err)
	ctx := logging.NewWriterContext(t.Output())
	cue, err := configcue.ResolveWorkspaceCuePath(ctx, "")
	require.NoError(t, err)
	if cue != "" {
		root, err := lewpath.Open(got.Prefix.Value())
		require.NoError(t, err)
		defer root.Close()
		ok, err := lewpath.New("modot.cue").IsFile(root)
		require.NoError(t, err)
		require.True(t, ok, "default %q has no modot.cue", got.Prefix.Value())
		return
	}
	ws, err := modfile.DetectWorkspace(ctx, "")
	require.NoError(t, err)
	require.Equal(t, ws.Root, got.Prefix.Value())
}
