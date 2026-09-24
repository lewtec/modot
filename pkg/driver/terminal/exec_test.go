package terminal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lucasew/workspaced/pkg/driver"
	_ "github.com/lucasew/workspaced/pkg/driver/exec/native"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestExecFactoryCompat(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "termfake")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx := logging.NewWriterContext(t.Output())

	missing := &execFactory{binary: "definitely-missing-term"}
	err := missing.CheckCompatibility(ctx)
	require.ErrorIs(t, err, driver.ErrIncompatible)

	okFactory := &execFactory{binary: "termfake"}
	require.NoError(t, okFactory.CheckCompatibility(ctx))

	envFirst := &execFactory{
		binary: "termfake",
		compat: func(ctx context.Context) error {
			return driver.RequireEnv(ctx, "WAYLAND_DISPLAY")
		},
	}
	t.Setenv("WAYLAND_DISPLAY", "")
	err = envFirst.CheckCompatibility(ctx)
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.Contains(t, err.Error(), "WAYLAND_DISPLAY")

	t.Setenv("WAYLAND_DISPLAY", "1")
	require.NoError(t, envFirst.CheckCompatibility(ctx))
}

func TestExecOpenArgs(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "termfake")
	body := "#!/bin/sh\nprintf '%s\\0' \"$@\" > \"$TERM_ARGS_FILE\"\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx := logging.NewWriterContext(t.Output())

	cases := []struct {
		name       string
		titleFlag  string
		commandAsE bool
		opts       Options
		want       []string
	}{
		{
			name:       "alacritty style",
			titleFlag:  "-T",
			commandAsE: true,
			opts:       Options{Title: "hi", Command: "echo", Args: []string{"a b"}},
			want:       []string{"-T", "hi", "-e", "echo", "a b"},
		},
		{
			name:       "bare command",
			titleFlag:  "--title",
			commandAsE: false,
			opts:       Options{Title: "title", Command: "nvim", Args: []string{"f"}},
			want:       []string{"--title", "title", "nvim", "f"},
		},
		{
			name:      "title only",
			titleFlag: "-T",
			opts:      Options{Title: "only"},
			want:      []string{"-T", "only"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := filepath.Join(dir, tc.name+".args")
			t.Setenv("TERM_ARGS_FILE", record)
			d := &execDriver{binary: "termfake", titleFlag: tc.titleFlag, commandAsE: tc.commandAsE}
			require.NoError(t, d.Open(ctx, tc.opts))
			got := splitNull(waitFile(t, record))
			require.Equal(t, tc.want, got)
		})
	}
}

func splitNull(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	parts := strings.Split(string(b), "\x00")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func waitFile(t *testing.T, path string) []byte {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		b, err := os.ReadFile(path)
		if err == nil {
			return b
		}
		if time.Now().After(deadline) {
			t.Fatalf("args file %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
