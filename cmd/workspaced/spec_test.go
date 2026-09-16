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
	assert.NotContains(t, got, "--profile-dir")
}

func TestRootUsageHasProfileDir(t *testing.T) {
	text, err := cmd.Usage[cmd.App[cli]]("workspaced")
	require.NoError(t, err)
	assert.Contains(t, text, "--profile-dir")
	assert.NotContains(t, text, "--cpuprofile")
	assert.NotContains(t, text, "--memprofile")
}

func TestHistoryDatabaseIsOnGroup(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "utils", "history", "--database", "/tmp/ws-hist.db", "list")
	require.NotNil(t, app.Args.Utils)
	require.NotNil(t, app.Args.Utils.History)
	require.NotNil(t, app.Args.Utils.History.DB.Value())
	assert.Equal(t, "/tmp/ws-hist.db", app.Args.Utils.History.DB.Value().URL())

	app = cmd.ParseOK[cmd.App[cli]](t, "utils", "history", "list", "--database", "/tmp/ws-hist2.db")
	require.NotNil(t, app.Args.Utils.History.DB.Value())
	assert.Equal(t, "/tmp/ws-hist2.db", app.Args.Utils.History.DB.Value().URL())
}

func TestSelfInstallForceSelected(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "self-install", "-f")
	require.NotNil(t, app.Args.Selfinstall)
	assert.True(t, app.Args.Selfinstall.Force.Value())
}

func TestLintFormatParse(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "codebase", "lint", "--format", "sarif")
	require.NotNil(t, app.Args.Codebase)
	require.NotNil(t, app.Args.Codebase.Lint)
	assert.Equal(t, "sarif", app.Args.Codebase.Lint.Format.Value().String())
}
