package db

import (
	"context"
	"path/filepath"
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

type ctxLeaf struct {
	url string
}

func (l *ctxLeaf) Run(ctx context.Context) error {
	d, err := OpenFromCtx(ctx)
	if err != nil {
		return err
	}
	l.url = d.conn.URL()
	return d.Close()
}

type ctxRoot struct {
	*Command `flatten:""`
	Leaf     *ctxLeaf
}

func TestOpenFromCtx(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ctx.db")
	app := cmd.ParseOK[cmd.App[ctxRoot]](t, "leaf", "--database", path)
	require.NoError(t, app.Run(t.Context()))
	require.NotNil(t, app.Args.Leaf)
	assert.Equal(t, path, app.Args.Leaf.url)
}
