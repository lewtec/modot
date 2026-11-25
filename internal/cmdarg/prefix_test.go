package cmdarg

import (
	"os"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/require"
)

func TestPrefixExpandsTilde(t *testing.T) {
	t.Parallel()
	var prefix Prefix
	require.NoError(t, prefix.Parse("~"))
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	require.Equal(t, home, prefix.Value())
}

func TestPrefixAcceptsSlashAndDot(t *testing.T) {
	t.Parallel()
	var root Prefix
	require.NoError(t, root.Parse("/"))
	require.Equal(t, "/", root.Value())
	var here Prefix
	require.NoError(t, here.Parse("."))
	require.Equal(t, ".", here.Value())
}

func TestPrefixFieldDefaults(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	type homeArgs struct {
		Prefix Prefix `long:"prefix" default:"~"`
	}
	parsedHome, err := cmd.Parse[homeArgs]()
	require.NoError(t, err)
	require.Equal(t, home, parsedHome.Prefix.Value())

	type dotArgs struct {
		Prefix Prefix `long:"prefix" default:"."`
	}
	parsedDot, err := cmd.Parse[dotArgs]()
	require.NoError(t, err)
	require.Equal(t, ".", parsedDot.Prefix.Value())

	type rootArgs struct {
		Prefix Prefix `long:"prefix" default:"/"`
	}
	parsedRoot, err := cmd.Parse[rootArgs]()
	require.NoError(t, err)
	require.Equal(t, "/", parsedRoot.Prefix.Value())
}

func TestPrefixRejectsMissing(t *testing.T) {
	t.Parallel()
	var prefix Prefix
	require.Error(t, prefix.Parse(t.TempDir()+"/missing"))
}
