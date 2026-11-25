package modfile

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

type stubHashProvider struct {
	id    string
	calls atomic.Int64
}

func (p *stubHashProvider) ID() string { return p.id }

func (p *stubHashProvider) ResolvePath(ctx context.Context, alias string, src SourceConfig, rel string, modulesBaseDir string) (string, error) {
	return "", fmt.Errorf("unused")
}

func (p *stubHashProvider) LockHash(ctx context.Context, alias string, src SourceConfig, modulesBaseDir string) (string, SourceConfig, error) {
	p.calls.Add(1)
	src.Ref = "main"
	src.URL = "https://example.test/" + alias
	return fmt.Sprintf("sha256:%s", alias), src, nil
}

func (p *stubHashProvider) Normalize(src SourceConfig) SourceConfig {
	src.Provider = p.id
	return src
}

func (p *stubHashProvider) EnrichRenovateDependency(dep *RenovateDependency, src LockedSource) {}
func (p *stubHashProvider) ConfigureFromSpec(cfg *SourceConfig, target string)                 {}
func (p *stubHashProvider) ResolveModuleRef(src SourceConfig, pathAndVersion string) (string, string, bool, error) {
	return "", "", false, nil
}
func (p *stubHashProvider) RehydrateLockedSource(dep RenovateDependency) (LockedSource, bool) {
	return LockedSource{}, false
}
func (p *stubHashProvider) LockLookupKeys(lock LockedSource) []string { return nil }
func (p *stubHashProvider) CanPersistLock(dep RenovateDependency, lock LockedSource) bool {
	return true
}
func (p *stubHashProvider) LockReusable(locked LockedSource) bool {
	return locked.Hash != ""
}
func (p *stubHashProvider) LockMatchesDesired(desired, locked LockedSource) bool { return true }

func registerStubHashProvider(t *testing.T, id string) *stubHashProvider {
	t.Helper()
	p := &stubHashProvider{id: id}
	prev, hadPrev := sourceProviders[id]
	RegisterSourceProvider(p)
	t.Cleanup(func() {
		if hadPrev {
			sourceProviders[id] = prev
			return
		}
		delete(sourceProviders, id)
	})
	return p
}

func testGroupCtx(t *testing.T) (*taskgroup.Session, context.Context) {
	t.Helper()
	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	t.Cleanup(func() {
		if err := g.Wait(); err != nil {
			t.Logf("group wait (may be expected in error tests): %v", err)
		}
	})
	return g, ctx
}

func TestPopulateSourceLockHashesSkipsExisting(t *testing.T) {
	t.Parallel()
	p := registerStubHashProvider(t, "stubhash-skip-existing")
	_, ctx := testGroupCtx(t)

	mod := &ModFile{Sources: map[string]SourceConfig{
		"alpha": {Provider: p.id, Repo: "o/alpha"},
		"beta":  {Provider: p.id, Repo: "o/beta"},
	}}
	entries := map[string]LockedSource{
		"alpha": {Provider: p.id, Repo: "o/alpha"},
		"beta":  {Provider: p.id, Repo: "o/beta", Hash: "sha256:keep"},
	}

	require.NoError(t, PopulateSourceLockHashes(ctx, mod, t.TempDir(), entries))
	require.Equal(t, int64(1), p.calls.Load(), "beta already hashed")
	require.Equal(t, "sha256:alpha", entries["alpha"].Hash)
	require.Equal(t, "main", entries["alpha"].Ref, "alpha missing resolved metadata: %+v", entries["alpha"])
	require.NotEmpty(t, entries["alpha"].URL, "alpha missing resolved metadata: %+v", entries["alpha"])
	require.Equal(t, "sha256:keep", entries["beta"].Hash)
}

// nestedInternetHashProvider is httpclient.WithProgress: LockHash Go's an
// Internet task and waits for it while still inside the Map child.
type nestedInternetHashProvider struct {
	stubHashProvider
}

func (p *nestedInternetHashProvider) LockHash(ctx context.Context, alias string, src SourceConfig, modulesBaseDir string) (string, SourceConfig, error) {
	done := make(chan struct{})
	taskgroup.Go(ctx, "fetch:"+alias, taskgroup.Internet, func(context.Context, *taskgroup.Status) error {
		close(done)
		return nil
	})
	select {
	case <-done:
		return p.stubHashProvider.LockHash(ctx, alias, src, modulesBaseDir)
	case <-ctx.Done():
		return "", src, ctx.Err()
	}
}

func TestPopulateSourceLockHashesNestedInternetDoesNotDeadlock(t *testing.T) {
	t.Parallel()
	p := &nestedInternetHashProvider{stubHashProvider{id: "stubhash-nested-net"}}
	prev, hadPrev := sourceProviders[p.id]
	RegisterSourceProvider(p)
	t.Cleanup(func() {
		if hadPrev {
			sourceProviders[p.id] = prev
			return
		}
		delete(sourceProviders, p.id)
	})

	// One Internet slot + more items than slots: the old PoolKind=Internet
	// Map held every slot while nested fetch waited → runtime deadlock.
	// Timeout must wrap New so acquire(g.ctx) unblocks instead of hanging the test.
	ctx, cancel := context.WithTimeout(logging.NewWriterContext(t.Output()), 2*time.Second)
	t.Cleanup(cancel)
	g, ctx := taskgroup.New(ctx, taskgroup.Limits{IO: 1, CPU: 1, Internet: 1})
	t.Cleanup(func() {
		if err := g.Wait(); err != nil {
			t.Logf("group wait: %v", err)
		}
	})

	mod := &ModFile{Sources: map[string]SourceConfig{
		"a": {Provider: p.id, Repo: "o/a"},
		"b": {Provider: p.id, Repo: "o/b"},
		"c": {Provider: p.id, Repo: "o/c"},
		"d": {Provider: p.id, Repo: "o/d"},
	}}
	entries := map[string]LockedSource{
		"a": {Provider: p.id, Repo: "o/a"},
		"b": {Provider: p.id, Repo: "o/b"},
		"c": {Provider: p.id, Repo: "o/c"},
		"d": {Provider: p.id, Repo: "o/d"},
	}

	require.NoError(t, PopulateSourceLockHashes(ctx, mod, t.TempDir(), entries))
	require.Equal(t, int64(4), p.calls.Load())
}

func TestPopulateSourceLockHashesParallel(t *testing.T) {
	t.Parallel()
	p := registerStubHashProvider(t, "stubhash-parallel")
	_, ctx := testGroupCtx(t)

	mod := &ModFile{Sources: map[string]SourceConfig{
		"one": {Provider: p.id, Repo: "o/one"},
		"two": {Provider: p.id, Repo: "o/two"},
	}}
	entries := map[string]LockedSource{
		"one": {Provider: p.id, Repo: "o/one"},
		"two": {Provider: p.id, Repo: "o/two"},
	}

	require.NoError(t, PopulateSourceLockHashes(ctx, mod, t.TempDir(), entries))
	require.Equal(t, int64(2), p.calls.Load())
	require.Equal(t, "sha256:one", entries["one"].Hash, "entries=%+v", entries)
	require.Equal(t, "sha256:two", entries["two"].Hash, "entries=%+v", entries)
}

func TestPopulateSourceLockHashesSkipsWhenComplete(t *testing.T) {
	t.Parallel()
	p := registerStubHashProvider(t, "stubhash-skip-all")
	_, ctx := testGroupCtx(t)

	mod := &ModFile{Sources: map[string]SourceConfig{
		"done": {Provider: p.id, Repo: "o/done"},
	}}
	entries := map[string]LockedSource{
		"done": {Provider: p.id, Repo: "o/done", Hash: "sha256:done"},
	}
	require.NoError(t, PopulateSourceLockHashes(ctx, mod, t.TempDir(), entries))
	require.Equal(t, int64(0), p.calls.Load())
}

func TestPopulateSourceLockHashesUnsupportedProvider(t *testing.T) {
	t.Parallel()
	_, ctx := testGroupCtx(t)
	mod := &ModFile{Sources: map[string]SourceConfig{
		"x": {Provider: "no-such-provider"},
	}}
	entries := map[string]LockedSource{
		"x": {Provider: "no-such-provider"},
	}
	err := PopulateSourceLockHashes(ctx, mod, t.TempDir(), entries)
	require.Error(t, err)
}
