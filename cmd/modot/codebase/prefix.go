package codebase

import (
	"log/slog"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/modfile"
	"github.com/lewtec/modot/internal/filespine"
	"github.com/lewtec/modot/internal/logging"
)

// Prefix is codebase --prefix. It embeds the data-directory flag.
// The default is the workspace that holds modot.cue, then the git root.
// StatePath, ConfigDir, and ModulesDir are paths inside that directory.
type Prefix struct {
	cmdarg.Prefix
}

func (Prefix) ArgDefault() string {
	// Flag defaults run during parse, before the command context exists.
	ctx := logging.NewRootContext(slog.Default())
	if cue, err := configcue.ResolveWorkspaceCuePath(ctx, ""); err == nil && cue != "" {
		if dir, err := parentDir(cue); err == nil {
			return dir
		}
	}
	ws, err := modfile.DetectWorkspace(ctx, "")
	if err != nil || ws == nil || ws.Root == "" {
		return "."
	}
	return ws.Root
}

// StatePath is the codebase state file, relative to the prefix root.
func (Prefix) StatePath() lewpath.Path {
	return lewpath.New(".modot", "state.json")
}

// ConfigDir is the direct config tree, relative to the prefix root.
func (Prefix) ConfigDir() lewpath.Path {
	return lewpath.New(".modot", "config")
}

// ModulesDir is the module tree, relative to the prefix root.
func (Prefix) ModulesDir() lewpath.Path {
	return lewpath.New("modules")
}

// directory is the host path of rel inside workspace.
// An existing directory is the opened root name. A missing one stays a relative Path on that root.
func directory(workspace *lewpath.Root, rel lewpath.Path) (string, error) {
	ok, err := rel.IsDir(workspace)
	if err != nil {
		return "", err
	}
	if !ok {
		return filespine.HostPath(workspace.Name(), rel), nil
	}
	opened, err := rel.OpenRoot(workspace)
	if err != nil {
		return "", err
	}
	name := opened.Name()
	err = opened.Close()
	return name, err
}

func parentDir(file string) (string, error) {
	opened, err := filespine.OpenDir(lewpath.New(file).Parent())
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
