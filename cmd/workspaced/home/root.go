package home

import ()

type Command struct {
	children `flatten:""`
}

func (Command) Description() string {
	return "Dotfiles and system state management"
}
