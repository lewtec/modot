package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	lewtest "github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeHelp(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "home", "--help")
	got := lewtest.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "Dotfiles and system state management")
	assert.Contains(t, got, "apply")
	assert.NotContains(t, got, "--cpuprofile")
}

func TestLintFormatParse(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "codebase", "lint", "--format", "sarif")
	require.NotNil(t, app.Args.Codebase)
	require.NotNil(t, app.Args.Codebase.Lint)
	assert.Equal(t, "sarif", app.Args.Codebase.Lint.Format.Value().String())
}