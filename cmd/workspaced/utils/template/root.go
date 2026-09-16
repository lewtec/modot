package template

import ()

type Command struct {
	Materialize *Materialize
}

func (Command) Description() string {
	return "Template management commands"
}
