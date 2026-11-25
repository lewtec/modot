package sourcecache

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/modot/internal/cmdctx"
	"github.com/lewtec/modot/internal/logging"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return logging.NewWriterContext(t.Output())
}

func TestEnsureCachedDirHitAndNoCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// UserHomeDir on some platforms also checks these; keep cache under home.
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	ctx := testCtx(t)
	var fetches atomic.Int32
	fetch := func(tmpDir string) error {
		fetches.Add(1)
		return os.WriteFile(filepath.Join(tmpDir, "marker"), []byte("v1"), 0o644)
	}

	dir1, err := EnsureCachedDir(ctx, "test", "key-a", fetch)
	require.NoError(t, err, "first ensure")
	require.Equal(t, int32(1), fetches.Load(), "fetches after miss")

	dir2, err := EnsureCachedDir(ctx, "test", "key-a", fetch)
	require.NoError(t, err, "second ensure")
	require.Equal(t, dir1, dir2, "cache dir changed")
	require.Equal(t, int32(1), fetches.Load(), "fetches after hit")

	// no-cache: re-fetch even though warm
	ctxNo := cmdctx.WithNoCache(ctx, true)
	dir3, err := EnsureCachedDir(ctxNo, "test", "key-a", fetch)
	require.NoError(t, err, "no-cache ensure")
	require.Equal(t, dir1, dir3, "dest path should be stable")
	require.Equal(t, int32(2), fetches.Load(), "fetches after no-cache")

	// no-cache + dry-run: do not re-fetch when warm
	ctxPlan := cmdctx.WithDryRun(ctxNo, true)
	_, err = EnsureCachedDir(ctxPlan, "test", "key-a", fetch)
	require.NoError(t, err, "dry-run no-cache")
	require.Equal(t, int32(2), fetches.Load(), "fetches after dry-run no-cache")
}
