package system

import ()

type Command struct {
	Apply *Apply
}

func (Command) Description() string {
	return "System apply tools"
}
