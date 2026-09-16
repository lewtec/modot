package config

import (
	"github.com/lucasew/workspaced/cmd/workspaced/configcmd"
)

type Command struct {
	Dump   *configcmd.Dump[configcmd.Home]
	Get    *configcmd.Get[configcmd.Home]
	Eval   *configcmd.Eval[configcmd.Home]
	Def    *configcmd.Def[configcmd.Home]
	Layers *configcmd.Layers[configcmd.Home]
}

func (Command) Description() string { return "Manage configuration" }
