package tool

import (
	_ "github.com/lucasew/workspaced/internal/tool/prelude"
)

type Command struct {
	List      *List
	Install   *Install
	Latest    *Latest
	Versions  *Versions
	Search    *Search
	Which     *Which
	With      *With
	Artifacts *Artifacts
}

func (Command) Description() string {
	return "Manage development tools"
}
