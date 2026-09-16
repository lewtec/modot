package driver

import ()

type Command struct {
	children `flatten:""`
}

func (Command) Description() string {
	return "Commands to interact with drivers"
}
