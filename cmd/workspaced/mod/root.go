package mod

import (
	_ "github.com/lucasew/workspaced/internal/modfile/sourceprovider/prelude"
)

type Command struct {
	Lock *Lock
	Tidy *Tidy
}

func (Command) Description() string {
	return "Manage module sources and lockfile"
}
