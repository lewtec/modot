package github

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGzStripsPrefix(t *testing.T) {
	t.Parallel()
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	content := []byte("hello module")
	if err := tw.WriteHeader(&tar.Header{
		Name: "repo-sha/subdir/file.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := extractTarGz(t.Context(), bytes.NewReader(gz.Bytes()), dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "subdir", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("got %q, want %q", got, content)
	}
	if _, err := os.Stat(filepath.Join(dest, "repo-sha")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("expected top-level prefix to be stripped")
	}
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	if err := tw.WriteHeader(&tar.Header{
		Name: "repo-sha/../../outside.txt",
		Mode: 0o644,
		Size: 3,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("bad")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := extractTarGz(t.Context(), bytes.NewReader(gz.Bytes()), dest); err == nil {
		t.Fatal("expected illegal path")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "outside.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("path traversal wrote outside dest")
	}
}
