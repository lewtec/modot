package workspace

type Command struct {
	Rotate     *Rotate
	Scratchpad *Scratchpad
	Next       *Next
}

func (Command) Description() string {
	return "Workspace management commands"
}
