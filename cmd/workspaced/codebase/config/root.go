package config

import (
	"github.com/lucasew/workspaced/cmd/workspaced/configcmd"
)

type Command struct {
	configcmd.Tree[configcmd.Codebase] `flatten:""`
}

func (Command) Description() string { return "Manage configuration" }
