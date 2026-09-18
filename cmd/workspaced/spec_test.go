package main

import (
	"io"
	"log/slog"
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
}

func TestRootUsageHasProfileDir(t *testing.T) {
	text, err := cmd.Usage[cmd.App[cli]]("workspaced")
	require.NoError(t, err)
	assert.Contains(t, text, "--profile-dir")
	assert.Contains(t, text, "--io")
	assert.Contains(t, text, "--cpu")
	assert.Contains(t, text, "--internet")
	assert.NotContains(t, text, "--cpuprofile")
	assert.NotContains(t, text, "--memprofile")
}

func TestSessionArgOverridesIO(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "--io", "9", "home", "--help")
	assert.Equal(t, 9, app.Args.IO.Value())
}

func TestHistoryDatabaseIsOnGroup(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "utils", "history", "--database", "/tmp/ws-hist.db", "list")
	require.NotNil(t, app.Args.Utils)
	require.NotNil(t, app.Args.Utils.History)
	require.NotNil(t, app.Args.Utils.History.Database.Value())
	assert.Equal(t, "/tmp/ws-hist.db", app.Args.Utils.History.Database.Value().URL())

	app = cmd.ParseOK[cmd.App[cli]](t, "utils", "history", "list", "--database", "/tmp/ws-hist2.db")
	assert.Equal(t, "/tmp/ws-hist2.db", app.Args.Utils.History.Database.Value().URL())
}

func TestSelfInstallForceSelected(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "self-install", "-f")
	require.NotNil(t, app.Args.Selfinstall)
	assert.True(t, app.Args.Selfinstall.Force.Value())
}

func TestCLISetupRestoresProcessLogger(t *testing.T) {
	lewtest.RestoreSlog(t)
	prev := processLogger
	t.Cleanup(func() { processLogger = prev })

	want := slog.Default()
	processLogger = want
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, (*cli)(nil).Setup())
	assert.Equal(t, want, slog.Default())
}

func TestLintFormatParse(t *testing.T) {
	app := cmd.ParseOK[cmd.App[cli]](t, "codebase", "lint", "--format", "sarif")
	require.NotNil(t, app.Args.Codebase)
	require.NotNil(t, app.Args.Codebase.Lint)
	assert.Equal(t, "sarif", app.Args.Codebase.Lint.Format.Value().String())
}

func TestOpenLazyOmitsBin(t *testing.T) {
	// Hermes shim and `open mise` docs: open lazy --home <tool> -- <args>
	app := cmd.ParseOK[cmd.App[cli]](t, "open", "lazy", "--home", "uv", "--", "run", "--project", "/tmp/hermes", "hermes")
	require.NotNil(t, app.Args.Open)
	require.NotNil(t, app.Args.Open.Lazy)
	assert.True(t, app.Args.Open.Lazy.Home.Value())
	assert.Empty(t, app.Args.Open.Lazy.Bin.Value())
}
