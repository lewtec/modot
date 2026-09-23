package codebase

import (
	"log/slog"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/workspaced/internal/cmdarg"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/modfile"
	"github.com/lucasew/workspaced/pkg/logging"
)

// Prefix is codebase --prefix. It embeds the data-directory flag.
// The default is the workspace that holds workspaced.cue, then the git root.
// StatePath is the codebase state file under that directory.
type Prefix struct {
	cmdarg.Prefix
}

func (Prefix) ArgDefault() string {
	// Flag defaults run during parse, before the command context exists.
	ctx := logging.NewRootContext(slog.Default())
	if cue, err := configcue.ResolveWorkspaceCuePath(ctx, ""); err == nil && cue != "" {
		return lewpath.New(cue).Parent().String()
	}
	ws, err := modfile.DetectWorkspace(ctx, "")
	if err != nil || ws == nil || ws.Root == "" {
		return "."
	}
	return ws.Root
}

// StatePath is <prefix>/.workspaced/state.json.
func (p Prefix) StatePath() string {
	return stateFile(p.Value())
}

func stateFile(dir string) string {
	return lewpath.New(dir, ".workspaced", "state.json").String()
}

var (
	_ cmd.Parser       = (*Prefix)(nil)
	_ cmd.Arg[string]  = (*Prefix)(nil)
	_ cmd.ArgDefaulter = Prefix{}
)
