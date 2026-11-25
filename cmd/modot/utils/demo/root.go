package demo

import ()

type Command struct {
	Debug    *Debug
	Progress *Progress
}

func (Command) Description() string {
	return "Demo commands"
}
