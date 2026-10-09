package githubutil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyAuthNilRequest(t *testing.T) {
	ApplyAuth(t.Context(), nil)
}

func TestApplyAuthSetsBearer(t *testing.T) {
	t.Setenv("LEWKIT_GITHUB_TOKEN_PROBE", "")
	t.Setenv("GH_TOKEN", "ghs_test_token")
	t.Setenv("GITHUB_TOKEN", "")

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)
	ApplyAuth(t.Context(), req)
	require.Equal(t, "Bearer ghs_test_token", req.Header.Get("Authorization"))
}

func TestApplyAuthOmitsHeaderWhenProbe(t *testing.T) {
	t.Setenv("LEWKIT_GITHUB_TOKEN_PROBE", "1")
	t.Setenv("GH_TOKEN", "ghs_test_token")
	t.Setenv("GITHUB_TOKEN", "ghs_test_token")

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)
	ApplyAuth(t.Context(), req)
	require.Empty(t, req.Header.Get("Authorization"))
}
