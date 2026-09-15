package archive

import (
	stdzip "archive/zip"
	"fmt"
	"io"
	"strings"

	lewfs "github.com/lewtec/lewkit/x/fs"
	xpath "github.com/lewtec/lewkit/x/path"
)

// RejectZipSlip reports [ErrIllegalPath] if any ZIP name is not a valid [io/fs] path.
func RejectZipSlip(r io.Reader) error {
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
