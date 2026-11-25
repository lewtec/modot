package codebase

import (
	pkg_config "github.com/lewtec/modot/cmd/modot/codebase/config"
)

type Command struct {
	Config   *pkg_config.Command
	Apply    *Apply
	Plan     *Plan
	Lint     *Lint
	Format   *Format
	Lsp      *Lsp
	CIStatus *CIStatus `cmd:"ci-status"`
}

func (Command) Description() string {
	return "Tools for analyzing and managing codebases"
}
