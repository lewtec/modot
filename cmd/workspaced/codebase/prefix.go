package codebase

import (
	"log/slog"
	"path/filepath"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lucasew/workspaced/internal/cmdarg"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/modfile"
	"github.com/lucasew/workspaced/pkg/logging"
)

// Prefix is codebase --prefix. It embeds the data-directory flag.
// The default is the workspace that holds workspaced.cue, then the git root.
// StatePath, ConfigDir, and ModulesDir are paths inside that directory.
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

// StatePath is the codebase state file, relative to the prefix root.
func (Prefix) StatePath() lewpath.Path {
	return lewpath.New(".workspaced", "state.json")
}

// ConfigDir is the direct config tree, relative to the prefix root.
func (Prefix) ConfigDir() lewpath.Path {
	return lewpath.New(".workspaced", "config")
}

// ModulesDir is the module tree, relative to the prefix root.
func (Prefix) ModulesDir() lewpath.Path {
	return lewpath.New("modules")
}

// directory opens rel under workspace when it exists.
// A missing directory is the root name plus the relative path.
func directory(workspace *lewpath.Root, rel lewpath.Path) (string, error) {
	ok, err := rel.IsDir(workspace)
	if err != nil {
		return "", err
	}
	if !ok {
		return filepath.Join(workspace.Name(), filepath.FromSlash(rel.String())), nil
	}
	opened, err := rel.OpenRoot(workspace)
	if err != nil {
		return "", err
	}
	name := opened.Name()
	err = opened.Close()
	return name, err
}

var (
	_ cmd.Parser       = (*Prefix)(nil)
	_ cmd.Arg[string]  = (*Prefix)(nil)
	_ cmd.ArgDefaulter = Prefix{}
)
