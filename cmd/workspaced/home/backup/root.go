package backup

import ()

type Command struct {
	RunCmd *Run `cmd:"run"`
}

func (Command) Description() string {
	return "Data backup and synchronization"
}
