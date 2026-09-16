package taskgroup

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgApplyOverridesPositiveFlags(t *testing.T) {
	got := cmd.ParseOK[Arg](t, "--io", "8", "--cpu", "2")
	base := Limits{IO: 4, CPU: 16, Internet: 4}
	lim := got.Apply(base)
	assert.Equal(t, 8, lim.IO)
	assert.Equal(t, 2, lim.CPU)
	assert.Equal(t, 4, lim.Internet)
}

func TestArgApplyKeepsBaseWhenFlagsOmitted(t *testing.T) {
	got := cmd.ParseOK[Arg](t)
	base := Limits{IO: 3, CPU: 5, Internet: 7}
	require.Equal(t, base, got.Apply(base))
}
