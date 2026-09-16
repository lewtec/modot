package clirun

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type group struct {
	children
}

type children struct {
	Leaf *leafCmd
}

type leafCmd struct {
	Force cmd.Flag `short:"f" long:"force"`
	ran   bool
}

func (l *leafCmd) Run(context.Context) error {
	l.ran = true
	return nil
}

func TestRunReachesEmbeddedLeaf(t *testing.T) {
	spec := cmd.ParseOK[group](t, "leaf", "-f")
	require.NoError(t, Run(t.Context(), &spec))
	require.NotNil(t, spec.Leaf)
	assert.True(t, spec.Leaf.Force.Value())
	assert.True(t, spec.Leaf.ran)
}

func TestRunGroupWithoutLeafIsUsage(t *testing.T) {
	var spec group
	err := Run(t.Context(), &spec)
	require.ErrorIs(t, err, cmd.ErrUsage)
}
