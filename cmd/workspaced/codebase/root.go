package codebase

import ()

type Command struct {
	children `flatten:""`
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
