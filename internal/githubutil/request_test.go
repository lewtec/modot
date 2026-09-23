package githubutil

import (
	"io"
	"net/http"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestNewAPIRequestHeaders(t *testing.T) {
	t.Setenv(githubTokenProbeEnv, githubTokenProbeVal)
	ctx := logging.NewWriterContext(io.Discard)
	req, err := NewAPIRequest(ctx, http.MethodGet, "https://api.github.com/repos/o/r/releases")
	require.NoError(t, err)
	require.Equal(t, UserAgent, req.Header.Get("User-Agent"))
	require.Equal(t, APIVersion, req.Header.Get("X-GitHub-Api-Version"))
	require.Empty(t, req.Header.Get("Authorization"), "Authorization with probe env")
}

func TestApplyAPIHeadersNilSafe(t *testing.T) {
	ApplyAPIHeaders(logging.NewWriterContext(io.Discard), nil)
}
