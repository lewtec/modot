package sourcecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/lewtec/modot/internal/atomicfile"
	"github.com/lewtec/modot/internal/cmdctx"
)

var (
	cacheLockMu sync.Mutex
	cacheLocks  = map[string]*sync.Mutex{}
)

func EnsureCachedDir(ctx context.Context, provider string, key string, fetch func(tmpDir string) error) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheRoot := filepath.Join(home, ".cache", "modot", "sources", provider)
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(key))
	dest := filepath.Join(cacheRoot, hex.EncodeToString(hash[:]))
	noCache := cmdctx.IsNoCache(ctx)

	if st, err := os.Stat(dest); err == nil && st.IsDir() && !noCache {
		slog.Debug("source cache hit", "provider", provider, "cache_dir", dest)
		return dest, nil
	}

	lock := keyLock(provider + "|" + key)
	lock.Lock()
	defer lock.Unlock()

	if st, err := os.Stat(dest); err == nil && st.IsDir() && !noCache {
		slog.Debug("source cache hit after wait", "provider", provider, "cache_dir", dest)
		return dest, nil
	}

	// Dry-run + no-cache: widen plan only; do not re-fetch.
	if noCache && cmdctx.IsDryRun(ctx) {
		if st, err := os.Stat(dest); err == nil && st.IsDir() {
			slog.Debug("no-cache: would re-fetch source (dry-run)", "provider", provider, "cache_dir", dest)
			return dest, nil
		}
	}

	if noCache {
		slog.Debug("no-cache: source cache miss", "provider", provider, "cache_dir", dest)
	} else {
		slog.Info("source cache miss", "provider", provider, "cache_dir", dest)
	}
	tmpDest := dest + ".tmp"
	if err := func() error { return os.RemoveAll(tmpDest) }(); err != nil {
		slog.Error("unexpected error", "op", "remove_all", "path", tmpDest, "error", err)
	}
	if err := os.RemoveAll(tmpDest); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tmpDest, 0755); err != nil {
		return "", err
	}
	slog.Info("source fetch start", "provider", provider, "tmp_dir", tmpDest)
	if err := fetch(tmpDest); err != nil {
		if err := func() error { return os.RemoveAll(tmpDest) }(); err != nil {
			slog.Error("unexpected error", "op", "remove_all", "path", tmpDest, "error", err)
		}
		return "", err
	}
	if err := atomicReplaceDir(dest, tmpDest); err != nil {
		if err := func() error { return os.RemoveAll(tmpDest) }(); err != nil {
			slog.Error("unexpected error", "op", "remove_all", "path", tmpDest, "error", err)
		}
		return "", err
	}
	slog.Info("source fetch done", "provider", provider, "cache_dir", dest)
	return dest, nil
}

func atomicReplaceDir(dest, tmpDir string) error {
	return atomicfile.ReplaceDir(dest, tmpDir)
}

func keyLock(key string) *sync.Mutex {
	cacheLockMu.Lock()
	defer cacheLockMu.Unlock()
	lock, ok := cacheLocks[key]
	if ok {
		return lock
	}
	lock = &sync.Mutex{}
	cacheLocks[key] = lock
	return lock
}
