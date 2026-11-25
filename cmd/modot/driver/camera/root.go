package camera

import ()

type Command struct {
	List    *List
	Capture *Capture
}

func (Command) Description() string {
	return "Camera capture management"
}
