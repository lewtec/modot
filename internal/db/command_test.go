package db

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type withStore struct {
	*Command `flatten:""`
}

func TestCommandInlineFlatten(t *testing.T) {
	got := cmd.ParseOK[withStore](t, "--database", "/tmp/inline.db")
	require.NotNil(t, got.Command)
	require.NotNil(t, got.Database.Value())
	assert.Equal(t, "/tmp/inline.db", got.Database.Value().URL())
}

func TestCommandDefaultURL(t *testing.T) {
	got := cmd.ParseOK[withStore](t)
	require.NotNil(t, got.Command)
	assert.Equal(t, (Arg{}).ArgDefault(), got.Database.Value().URL())
}
