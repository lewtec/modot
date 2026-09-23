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
		t.Fatal("state path mismatch")
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
	cue, err := configcue.ResolveWorkspaceCuePath(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if cue != "" {
		root, err := lewpath.Open(got.Prefix.Value())
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close()
		ok, err := lewpath.New("workspaced.cue").IsFile(root)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("default %q has no workspaced.cue", got.Prefix.Value())
		}
		return
	}
	ws, err := modfile.DetectWorkspace(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prefix.Value() != ws.Root {
		t.Fatalf("default = %q want %q", got.Prefix.Value(), ws.Root)
	}
}
