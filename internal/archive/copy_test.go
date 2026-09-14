package archive

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	xpath "github.com/lewtec/lewkit/x/path"
	xtest "github.com/lewtec/lewkit/x/test"
)

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
