package archive

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	lewfs "github.com/lewtec/lewkit/x/fs"
	xpath "github.com/lewtec/lewkit/x/path"
	xtest "github.com/lewtec/lewkit/x/test"
)

type onlyReader struct{ io.Reader }

func TestCopyFSWritesTree(t *testing.T) {
	t.Parallel()
	src := fstest.MapFS{
		"a.txt":     {Data: []byte("hello")},
		"bin/tool":  {Data: []byte("#!/bin/sh\n"), Mode: 0o755},
		"dir/b.txt": {Data: []byte("nested")},
	}
	dest := t.TempDir()
	root, err := xpath.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	xtest.CloseOnCleanup(t, root)
	if err := CopyFS(root, src); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("a.txt=%q", got)
	}
	got, err = os.ReadFile(filepath.Join(dest, "dir", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "nested" {
		t.Fatalf("dir/b.txt=%q", got)
	}
	st, err := os.Stat(filepath.Join(dest, "bin", "tool"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected executable, mode=%o", st.Mode().Perm())
	}
}

func TestExtractUDFNeedReadAt(t *testing.T) {
	t.Parallel()
	err := ExtractUDF(onlyReader{strings.NewReader("x")}, t.TempDir())
	if !errors.Is(err, lewfs.ErrNeedReadAt) {
		t.Fatalf("err=%v, want ErrNeedReadAt", err)
	}
}

func TestExtractUDFInvalidVolume(t *testing.T) {
	t.Parallel()
	err := ExtractUDF(bytes.NewReader(make([]byte, 4096)), t.TempDir())
	if err == nil {
		t.Fatal("expected invalid volume")
	}
	if errors.Is(err, lewfs.ErrNeedReadAt) {
		t.Fatal("byte reader is a ReaderAt")
	}
}

func TestExtractWIMNeedReadAt(t *testing.T) {
	t.Parallel()
	err := ExtractWIM(onlyReader{strings.NewReader("x")}, t.TempDir(), 1)
	if !errors.Is(err, lewfs.ErrNeedReadAt) {
		t.Fatalf("err=%v, want ErrNeedReadAt", err)
	}
}

func TestExtractFSRejectsInvalidName(t *testing.T) {
	t.Parallel()
	src := invalidNameFS{}
	if err := ExtractFS(t.TempDir(), src); err == nil || !errors.Is(err, ErrIllegalPath) {
		t.Fatalf("err=%v, want ErrIllegalPath", err)
	}
}

type invalidNameFS struct{}

func (invalidNameFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &invalidDir{}, nil
	}
	return nil, fs.ErrNotExist
}

type invalidDir struct{ done bool }

func (invalidDir) Stat() (fs.FileInfo, error) { return dummyInfo{name: ".", dir: true}, nil }
func (invalidDir) Read([]byte) (int, error)   { return 0, fs.ErrInvalid }
func (invalidDir) Close() error               { return nil }
func (d *invalidDir) ReadDir(int) ([]fs.DirEntry, error) {
	if d.done {
		return nil, io.EOF
	}
	d.done = true
	return []fs.DirEntry{dummyEntry{name: ".."}}, nil
}

type dummyEntry struct{ name string }

func (d dummyEntry) Name() string               { return d.name }
func (d dummyEntry) IsDir() bool                { return false }
func (d dummyEntry) Type() fs.FileMode          { return 0 }
func (d dummyEntry) Info() (fs.FileInfo, error) { return dummyInfo{name: d.name}, nil }

type dummyInfo struct {
	name string
	dir  bool
}

func (d dummyInfo) Name() string      { return d.name }
func (d dummyInfo) Size() int64       { return 0 }
func (d dummyInfo) Mode() fs.FileMode { return 0 }
func (d dummyInfo) ModTime() time.Time {
	return time.Time{}
}
func (d dummyInfo) IsDir() bool { return d.dir }
func (d dummyInfo) Sys() any    { return nil }
