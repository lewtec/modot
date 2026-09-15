package archive

import (
	stdzip "archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/lewtec/lewkit/x/compression"
	_ "github.com/lewtec/lewkit/x/compression/prelude"
	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/squashfs"
	tarfs "github.com/lewtec/lewkit/x/fs/tar"
	zipfs "github.com/lewtec/lewkit/x/fs/zip"
	xpath "github.com/lewtec/lewkit/x/path"
)

// ExtractZip unpacks a ZIP via [zipfs.Open] and [lewfs.Copy].
func ExtractZip(ctx context.Context, r io.Reader, destDir string) error {
	if err := rejectZipSlip(r); err != nil {
		return err
	}
	z, err := zipfs.Open(r)
	if err != nil {
		return err
	}
	return ExtractFS(ctx, destDir, z)
}

// ExtractTar unpacks a tar listing via [tarfs.Files] and [lewfs.Copy].
// Compressed wrappers are unwrapped with [compression.Detect].
func ExtractTar(ctx context.Context, r io.Reader, destDir string) error {
	plain, err := unwrapCompression(r)
	if err != nil {
		return err
	}
	return CopyTo(ctx, destDir, tarfs.Files(plain))
}

// ExtractSquashFS unpacks a SquashFS image via [squashfs.Open] and [lewfs.Copy].
func ExtractSquashFS(ctx context.Context, r io.Reader, destDir string) error {
	img, err := squashfs.Open(r)
	if err != nil {
		return err
	}
	return ExtractFS(ctx, destDir, img)
}

func unwrapCompression(r io.Reader) (io.Reader, error) {
	name := nameOf(r)
	if ra, ok := r.(io.ReaderAt); ok {
		if c, ok := compression.Detect(name, peekAt(ra, 16)); ok {
			return decompress(c, io.NewSectionReader(ra, 0, 1<<63-1))
		}
		return r, nil
	}
	hdr, src, err := peekCopy(r, 16)
	if err != nil {
		return nil, err
	}
	if c, ok := compression.Detect(name, hdr); ok {
		return decompress(c, src)
	}
	return src, nil
}

func decompress(c compression.Codec, r io.Reader) (io.Reader, error) {
	d, ok := c.(compression.Decompressor)
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: "", Err: fs.ErrInvalid}
	}
	cr, err := d.Reader(r)
	if err != nil {
		return nil, err
	}
	defer cr.Close()
	raw, err := io.ReadAll(cr)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
}

func nameOf(r io.Reader) string {
	st, ok := r.(interface {
		Stat() (fs.FileInfo, error)
	})
	if !ok {
		return ""
	}
	fi, err := st.Stat()
	if err != nil {
		return ""
	}
	return fi.Name()
}

func peekAt(ra io.ReaderAt, n int) []byte {
	b := make([]byte, n)
	got, err := ra.ReadAt(b, 0)
	if got == 0 && err != nil {
		return nil
	}
	return b[:got]
}

func peekCopy(r io.Reader, n int) ([]byte, io.Reader, error) {
	b := make([]byte, n)
	got, err := io.ReadFull(r, b)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		return nil, nil, err
	}
	hdr := b[:got]
	return hdr, io.MultiReader(bytes.NewReader(hdr), r), nil
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
