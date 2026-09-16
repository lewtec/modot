package nix

import (
	"errors"
)

var (
	ErrNoFlakeRef    = errors.New("no flake reference provided")
	ErrNoBinaryFound = errors.New("no binary found")
)

type Command struct {
	Build     *Build
	Deploy    *Deploy
	GcCleanup *GcCleanup `cmd:"gc-cleanup"`
	Rbuild    *Rbuild
	Rrun      *Rrun
	RunCmd    *Run `cmd:"run"`
}

func (Command) Description() string {
	return "Nix operations"
}
