package experiments

import ()

type Command struct {
	children `flatten:""`
	Cue      *Cue `cmd:"cue"`
}

func (Command) Description() string {
	return "Experimental features and prototypes"
}
