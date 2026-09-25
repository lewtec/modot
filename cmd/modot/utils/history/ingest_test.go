package history

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/require"
)

func TestIngestSubcommands(t *testing.T) {
	bash := cmd.ParseOK[Command](t, "ingest", "bash")
	require.NotNil(t, bash.Ingest)
	require.NotNil(t, bash.Ingest.Bash)
	require.Nil(t, bash.Ingest.Atuin)
	require.Nil(t, bash.Ingest.Workspaced)

	atuin := cmd.ParseOK[Command](t, "ingest", "atuin")
	require.NotNil(t, atuin.Ingest.Atuin)

	old := cmd.ParseOK[Command](t, "ingest", "workspaced")
	require.NotNil(t, old.Ingest.Workspaced)

	err := cmd.ParseErr[Command](t, "ingest", "atuin-history")
	require.Error(t, err)
}
