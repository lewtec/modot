package workspace

import (
	"github.com/lewtec/lewkit/x/cmd"
)

type Command struct {
	Move       cmd.Flag `long:"move" help:"Move container to workspace"`
	Rotate     *Rotate
	Scratchpad *Scratchpad
	Next       *Next
}

func (Command) Description() string {
	return "Workspace management commands"
}
