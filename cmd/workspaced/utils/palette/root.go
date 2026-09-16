package palette

import ()

type Command struct {
	children `flatten:""`
}

func (Command) Description() string {
	return "Color palette generation and management"
}
