package shell

import ()

type Command struct {
	Init *Init
}

func (Command) Description() string {
	return "Shell integration commands"
}
