package open

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazyBinOptional(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		args    []string
		home    bool
		tool    string
		bin     string
		toolArg []string
	}{
		{
			name:    "omit bin defaults to tool",
			args:    []string{"--home", "uv", "--", "run", "--project", "/tmp/hermes", "hermes"},
			home:    true,
			tool:    "uv",
			bin:     "uv",
			toolArg: []string{"run", "--project", "/tmp/hermes", "hermes"},
		},
		{
			name:    "explicit bin",
			args:    []string{"--home", "--bin", "uvx", "uv", "--", "--version"},
			home:    true,
			tool:    "uv",
			bin:     "uvx",
			toolArg: []string{"--version"},
		},
		{
			name:    "tool only",
			args:    []string{"mise", "--", "version"},
			tool:    "mise",
			bin:     "mise",
			toolArg: []string{"version"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cmd.ParseOK[Lazy](t, tc.args...)
			assert.Equal(t, tc.home, got.Home.Value())
			assert.Equal(t, tc.tool, got.tool.Value())
			assert.Equal(t, tc.bin, got.binName(got.tool.Value()))
			assert.Equal(t, tc.toolArg, cmd.Values(got.args))
		})
	}
}

func TestLazyRequiresTool(t *testing.T) {
	t.Parallel()
	err := cmd.ParseErr[Lazy](t, "--home", "--bin", "uv", "--", "--version")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "argument")
}
