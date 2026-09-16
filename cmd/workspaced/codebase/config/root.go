package config

import (
	"github.com/lucasew/workspaced/cmd/workspaced/configcmd"
)

type Command struct {
	Dump   *configcmd.Dump[configcmd.Codebase]
	Get    *configcmd.Get[configcmd.Codebase]
	Eval   *configcmd.Eval[configcmd.Codebase]
	Def    *configcmd.Def[configcmd.Codebase]
	Layers *configcmd.Layers[configcmd.Codebase]
}

func (Command) Description() string { return "Manage configuration" }
