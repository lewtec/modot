package archive

import (
	stdzip "archive/zip"
	"fmt"
	"io"
	"strings"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/squashfs"
	tarfs "github.com/lewtec/lewkit/x/fs/tar"
	zipfs "github.com/lewtec/lewkit/x/fs/zip"
	xpath "github.com/lewtec/lewkit/x/path"
)

// ExtractZip unpacks a ZIP via [zipfs.Open] into destDir.
func ExtractZip(r io.Reader, destDir string) error {
	if err := rejectZipSlip(r); err != nil {
		return err
	}
	z, err := zipfs.Open(r)
	if err != nil {
		return err
	}
	return ExtractFS(destDir, z)
}

// ExtractTar unpacks a tar via [tarfs.Open] into destDir.
// Compressed wrappers (.gz, .xz, .zst, …) are chosen by name or magic
// in the process-wide [github.com/lewtec/lewkit/x/compression] registry.
func ExtractTar(r io.Reader, destDir string) error {
	t, err := tarfs.Open(r)
	if err != nil {
		return err
	}
	return ExtractFS(destDir, t)
}

// ExtractSquashFS unpacks a SquashFS image via [squashfs.Open] into destDir.
func ExtractSquashFS(r io.Reader, destDir string) error {
	img, err := squashfs.Open(r)
	if err != nil {
		return err
	}
	return ExtractFS(destDir, img)
}

func rejectZipSlip(r io.Reader) error {
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return err
	}
	size, err := lewfs.Size("open", r)
	if err != nil {
		return err
	}
	zr, err := stdzip.NewReader(ra, size)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		name := strings.TrimSuffix(f.Name, "/")
		p := xpath.New(name)
		if name != "" && name != "." && (!p.Valid() || p.IsAbs()) {
			return fmt.Errorf("%w: %s", ErrIllegalPath, f.Name)
		}
	}
	return nil
}
