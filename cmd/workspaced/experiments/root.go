package experiments

import (
	pkg_demo "github.com/lucasew/workspaced/cmd/workspaced/experiments/demo"
)

type Command struct {
	Demo *pkg_demo.Command
	Cue  *Cue `cmd:"cue"`
}

func (Command) Description() string {
	return "Experimental features and prototypes"
}
