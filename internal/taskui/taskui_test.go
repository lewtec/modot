package taskui

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestRunCreatesSession(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	err := Run(ctx, func(ctx context.Context) error {
		require.NotNil(t, taskgroup.FromContext(ctx))
		return nil
	})
	require.NoError(t, err)
}
