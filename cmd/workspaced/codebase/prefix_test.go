package codebase

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/modfile"
	_ "github.com/lucasew/workspaced/pkg/driver/exec/native"
	"github.com/lucasew/workspaced/pkg/logging"
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
	want := lewpath.New(".workspaced", "state.json")
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
		ok, err := lewpath.New("workspaced.cue").IsFile(root)
		require.NoError(t, err)
		require.True(t, ok, "default %q has no workspaced.cue", got.Prefix.Value())
		return
	}
	ws, err := modfile.DetectWorkspace(ctx, "")
	require.NoError(t, err)
	require.Equal(t, ws.Root, got.Prefix.Value())
}
