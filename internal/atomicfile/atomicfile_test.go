package atomicfile

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBytesSuccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	require.NoError(t, WriteBytes(path, []byte("hello"), 0o644))
	assertNoTmpLeft(t, dir, filepath.Base(path))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hello", string(got))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "mode=%o", info.Mode().Perm())
}

func TestWriteString(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "s.txt")
	require.NoError(t, WriteString(path, "hi", 0o600))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hi", string(got))
}

func TestWriteFailureKeepsExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.bin")
	require.NoError(t, os.WriteFile(path, []byte("good"), 0o644))
	r := io.MultiReader(
		bytes.NewReader([]byte("partial-")),
		errReader{errors.New("boom")},
	)
	require.Error(t, Write(path, r, 0o644))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "good", string(got), "prior content lost")
	assertNoTmpLeft(t, dir, filepath.Base(path))
}

func TestWriteFailureLeavesNoFinal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "new.bin")
	require.Error(t, Write(path, errReader{errors.New("boom")}, 0o644))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, fs.ErrNotExist, "final path should not exist")
	assertNoTmpLeft(t, dir, filepath.Base(path))
}

func TestWritePNG(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "shot.png")
	require.NoError(t, os.WriteFile(path, []byte("truncated"), 0o644))
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	require.NoError(t, WritePNG(path, img))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 8)
	require.Equal(t, "\x89PNG\r\n\x1a\n", string(raw[:8]))
}

func TestCreateCommitEncode(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "shot.png")
	require.NoError(t, os.WriteFile(path, []byte("truncated"), 0o644))
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	f, err := Create(path, 0o644)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Abort(), "abort")
	}()
	require.NotEqual(t, SiblingTemp(path), f.Name(), "Create should use a unique temp name, not SiblingTemp")
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Commit())
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 8)
	require.Equal(t, "\x89PNG\r\n\x1a\n", string(raw[:8]))
}

func TestCreateSibling(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "out.txt")
	f, err := CreateSibling(path, 0o600)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Abort(), "abort")
	}()
	require.Equal(t, SiblingTemp(path), f.Name())
	_, err = io.WriteString(f, "x")
	require.NoError(t, err)
	require.NoError(t, f.Commit())
	_, err = os.Stat(SiblingTemp(path))
	require.ErrorIs(t, err, fs.ErrNotExist, "sibling tmp left")
}

func TestCreateMode(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "secret.json")
	f, err := Create(path, 0o600)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Abort(), "abort")
	}()
	_, err = io.WriteString(f, `{"a":1}`)
	require.NoError(t, err)
	require.NoError(t, f.Commit())
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "mode=%o", info.Mode().Perm())
}

func TestInstallChmod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tmp := filepath.Join(dir, "x.tmp")
	dest := filepath.Join(dir, "x")
	require.NoError(t, os.WriteFile(tmp, []byte("bin"), 0o644))
	require.NoError(t, Install(tmp, dest, 0o755))
	info, err := os.Stat(dest)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "mode=%o", info.Mode().Perm())
}

func TestReplaceDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dest := filepath.Join(root, "v1")
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "old"), []byte("old"), 0o644))
	tmp := filepath.Join(root, "v1.tmp")
	require.NoError(t, os.MkdirAll(tmp, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "new"), []byte("new"), 0o644))
	require.NoError(t, ReplaceDir(dest, tmp))
	_, err := os.Stat(filepath.Join(dest, "new"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "old"))
	require.ErrorIs(t, err, fs.ErrNotExist, "old content still present")
}

func assertNoTmpLeft(t *testing.T, dir, base string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	prefix := base + ".tmp"
	for _, e := range entries {
		name := e.Name()
		require.False(t, name == prefix || strings.HasPrefix(name, prefix+"-"), "tmp left behind: %s", name)
	}
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
