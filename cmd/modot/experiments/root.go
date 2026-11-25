package experiments

import (
	pkg_demo "github.com/lewtec/modot/cmd/modot/experiments/demo"
)

type Command struct {
	Demo *pkg_demo.Command
	Cue  *Cue `cmd:"cue"`
}

func (Command) Description() string {
	return "Experimental features and prototypes"
}
