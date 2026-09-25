package backup

import (
	"testing"

	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestGitRepoSyncActionHasHEAD(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	action := GitRepoSyncAction{Src: t.TempDir()}

	require.NoError(t, action.run(ctx, "init", "--quiet"), "git init")

	hasHEAD, err := action.hasHEAD(ctx)
	require.NoError(t, err, "check unborn HEAD")
	require.False(t, hasHEAD, "unborn repository unexpectedly has HEAD")

	require.NoError(t, action.run(ctx,
		"-c", "user.email=test@example.com",
		"-c", "user.name=Test User",
		"commit", "--quiet", "--allow-empty", "-m", "initial"), "create initial commit")

	hasHEAD, err = action.hasHEAD(ctx)
	require.NoError(t, err, "check committed HEAD")
	require.True(t, hasHEAD, "committed repository has no HEAD")
}
