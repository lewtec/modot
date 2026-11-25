package icons

import ()

type Command struct {
	Generate *Generate
}

func (Command) Description() string {
	return "Icon theme generation utilities"
}
