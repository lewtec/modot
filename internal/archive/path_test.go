package archive

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathWithinDest(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.True(t, PathWithinDest(dest, dest), "dest should be within itself")
	require.True(t, PathWithinDest(dest, filepath.Join(dest, "a", "b")), "nested should be within")
	require.False(t, PathWithinDest(dest, filepath.Join(dest, "..", "escape")), "parent escape should fail")
}

func TestJoinWithinRejectsTraversal(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	for _, name := range []string{"../outside", "foo/../../outside", "/abs"} {
		_, err := JoinWithin(dest, name)
		require.Error(t, err, "expected error for %q", name)
	}
	got, err := JoinWithin(dest, "ok/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dest, "ok", "file.txt"), got)
}

func TestWriteMemberRemovesPartial(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	r := io.MultiReader(
		bytes.NewReader([]byte("partial-")),
		errReader{errors.New("boom")},
	)
	require.Error(t, WriteMember(path, 0o644, r))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, fs.ErrNotExist, "partial still present")
}

func TestWriteMemberSuccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bin", "tool")
	require.NoError(t, WriteMember(path, 0o755, bytes.NewReader([]byte("#!/bin/sh\n"))))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o111, "expected executable, mode=%o", info.Mode().Perm())
}

func TestSymlinkTargetWithin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	require.True(t, SymlinkTargetWithin(dir, link, "sibling.txt"), "in-dest relative should be ok")
	require.False(t, SymlinkTargetWithin(dir, link, "../../outside"), "escaping relative should fail")
	require.False(t, SymlinkTargetWithin(dir, link, ""), "empty should fail")
}

func TestResolveWithin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	nested := filepath.Join(dir, "a")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	target := filepath.Join(nested, "f.txt")
	got, err := ResolveWithin(dir, target)
	require.NoError(t, err)
	if got != target && filepath.Clean(got) != filepath.Clean(target) {
		// EvalSymlinks may rewrite; still under dest
		require.True(t, PathWithinDest(dir, got), "resolved outside: %q", got)
	}
	// Symlink parent that escapes: dir/sub -> /tmp or parent
	escape := filepath.Join(dir, "escape-link")
	require.NoError(t, os.Symlink("..", escape))
	bad := filepath.Join(escape, "x")
	_, err = ResolveWithin(dir, bad)
	require.Error(t, err)
	if !errors.Is(err, ErrIllegalPath) {
		// EvalSymlinks may fail first; still require a non-nil error above.
		t.Logf("got err %v (ok if non-nil)", err)
	}
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
