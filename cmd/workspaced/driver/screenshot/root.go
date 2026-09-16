package screenshot

import ()

type Command struct {
	All    *All
	Full   *Full
	Output *Output
	Window *Window
	Select *Select
}

func (Command) Description() string {
	return "Screen capture management"
}
