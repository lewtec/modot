package github

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	lewfs "github.com/lewtec/lewkit/x/fs"
	tarfs "github.com/lewtec/lewkit/x/fs/tar"
	lewpath "github.com/lewtec/lewkit/x/path"
	lewtest "github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"

	"github.com/lucasew/workspaced/internal/archive"
)

func copyTar(t *testing.T, r io.Reader, dest string) error {
	t.Helper()
	root, err := lewpath.Open(dest)
	if err != nil {
		return err
	}
	lewtest.CloseOnCleanup(t, root)
	tfs, err := tarfs.Open(t.Context(), r)
	if err != nil {
		return err
	}
	if err := lewfs.Copy(t.Context(), root, lewfs.Walk(t.Context(), tfs, nil)); err != nil {
		return err
	}
	return archive.StripTopLevelDir(dest)
}

func TestExtractTarGzStripsPrefix(t *testing.T) {
	t.Parallel()
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	content := []byte("hello module")
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "repo-sha/subdir/file.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, err = zw.Write(raw.Bytes())
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	dest := t.TempDir()
	require.NoError(t, copyTar(t, bytes.NewReader(gz.Bytes()), dest))
	got, err := os.ReadFile(filepath.Join(dest, "subdir", "file.txt"))
	require.NoError(t, err)
	require.Equal(t, content, got)
	_, err = os.Stat(filepath.Join(dest, "repo-sha"))
	require.ErrorIs(t, err, fs.ErrNotExist, "expected top-level prefix to be stripped")
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "repo-sha/../../outside.txt",
		Mode: 0o644,
		Size: 3,
	}))
	_, err := tw.Write([]byte("bad"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, err = zw.Write(raw.Bytes())
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	dest := t.TempDir()
	err = copyTar(t, bytes.NewReader(gz.Bytes()), dest)
	require.Error(t, err, "expected illegal path")
	_, err = os.Stat(filepath.Join(filepath.Dir(dest), "outside.txt"))
	require.ErrorIs(t, err, fs.ErrNotExist, "path traversal wrote outside dest")
}
