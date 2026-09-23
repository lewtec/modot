package codebase

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/modfile"
	_ "github.com/lucasew/workspaced/pkg/driver/exec/native"
	"github.com/lucasew/workspaced/pkg/logging"
)

func TestPrefixFlagWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	type args struct {
		Prefix Prefix `long:"prefix"`
	}
	got, err := cmd.Parse[args]("--prefix", dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prefix.Value() != dir {
		t.Fatalf("value = %q", got.Prefix.Value())
	}
	want := lewpath.New(".workspaced", "state.json")
	if got.Prefix.StatePath() != want {
		t.Fatalf("state = %s", got.Prefix.StatePath())
	}
}

func TestPrefixDefaultIsWorkspaceRoot(t *testing.T) {
	t.Parallel()
	type args struct {
		Prefix Prefix `long:"prefix"`
	}
	got, err := cmd.Parse[args]()
	if err != nil {
		t.Fatal(err)
	}
	ctx := logging.NewWriterContext(t.Output())
	want := "."
	if cue, err := configcue.ResolveWorkspaceCuePath(ctx, ""); err == nil && cue != "" {
		want = lewpath.New(cue).Parent().String()
	} else if ws, err := modfile.DetectWorkspace(ctx, ""); err == nil && ws != nil && ws.Root != "" {
		want = ws.Root
	}
	if got.Prefix.Value() != want {
		t.Fatalf("default = %q want %q", got.Prefix.Value(), want)
	}
}
