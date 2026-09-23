package githubutil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestResolveTokenSTOP(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", githubTokenStop)
	t.Setenv(githubTokenProbeEnv, "")
	ctx := logging.NewWriterContext(io.Discard)
	got := resolveToken(ctx)
	require.Empty(t, got, "resolveToken with GITHUB_TOKEN=STOP")
}

func TestResolveTokenFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghs_test_token")
	t.Setenv(githubTokenProbeEnv, "")
	ctx := logging.NewWriterContext(io.Discard)
	got := resolveToken(ctx)
	require.Equal(t, "ghs_test_token", got)
}

func TestResolveTokenSTOPNotUsedAsBearer(t *testing.T) {
	// ApplyAuth goes through Token (sync.Once). Exercise resolveToken + header
	// policy without relying on process-global Token cache.
	t.Setenv("GITHUB_TOKEN", githubTokenStop)
	t.Setenv(githubTokenProbeEnv, "")
	ctx := logging.NewWriterContext(io.Discard)
	require.Empty(t, resolveToken(ctx))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)
	// Mirror ApplyAuth's rule: only set Authorization when token is non-empty.
	if tok := resolveToken(ctx); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	require.Empty(t, req.Header.Get("Authorization"), "STOP must not be sent as Bearer")
}

func TestTokenProbeEnvSkipsResolution(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv(githubTokenProbeEnv, githubTokenProbeVal)
	ctx := logging.NewWriterContext(io.Discard)
	// Token short-circuits on probe env before Once/env/gh.
	require.Empty(t, Token(ctx), "Token with probe env")
}

func TestResolveGHBinaryUsesLocatorWhenPATHMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty PATH: no gh
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv(githubTokenProbeEnv, "")

	want := filepath.Join(t.TempDir(), "gh")
	SetGHLocator(func(ctx context.Context) (string, error) {
		return want, nil
	})
	t.Cleanup(func() { SetGHLocator(nil) })

	ctx := logging.NewWriterContext(io.Discard)
	got, err := resolveGHBinary(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestResolveGHBinaryLocatorError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv(githubTokenProbeEnv, "")

	boom := errors.New("ensure failed")
	SetGHLocator(func(ctx context.Context) (string, error) {
		return "", boom
	})
	t.Cleanup(func() { SetGHLocator(nil) })

	ctx := logging.NewWriterContext(io.Discard)
	_, err := resolveGHBinary(ctx)
	require.ErrorIs(t, err, boom)
}

func TestResolveGHBinaryNoLocator(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv(githubTokenProbeEnv, "")
	SetGHLocator(nil)

	ctx := logging.NewWriterContext(io.Discard)
	_, err := resolveGHBinary(ctx)
	require.ErrorIs(t, err, errGHNotFound)
}
