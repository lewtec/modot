package apply

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTempDconfIni_UniqueAndContents(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	const body = "[org/gnome/desktop/interface]\ncolor-scheme='prefer-dark'\n\n"
	p1, err := writeTempDconfIni(ctx, body)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(p1), "remove %s", p1)
	})

	p2, err := writeTempDconfIni(ctx, body)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(p2), "remove %s", p2)
	})

	require.NotEqual(t, p2, p1, "expected unique temp paths")
	require.NotEqual(t, "workspaced-dconf.ini", filepath.Base(p1), "still using fixed temp name: %q", p1)
	require.Contains(t, filepath.Base(p1), "workspaced-dconf-")

	got, err := os.ReadFile(p1)
	require.NoError(t, err)
	require.Equal(t, body, string(got))

	info, err := os.Stat(p1)
	require.NoError(t, err)
	require.Zero(t, info.Mode().Perm()&0o077, "temp file should not be group/other accessible: mode %o", info.Mode().Perm())
}

func TestWriteTempDconfIni_ConcurrentNoCollision(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	const n = 16
	paths := make([]string, n)
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			p, err := writeTempDconfIni(ctx, strings.Repeat("x", i+1))
			if err != nil {
				errCh <- err
				return
			}
			paths[i] = p
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		for _, p := range paths {
			if p != "" {
				assert.NoError(t, os.Remove(p), "remove %s", p)
			}
		}
	})
	seen := map[string]struct{}{}
	for _, p := range paths {
		require.NotEmpty(t, p, "empty path from concurrent write")
		require.NotContains(t, seen, p, "duplicate temp path under concurrency: %q", p)
		seen[p] = struct{}{}
	}
}
