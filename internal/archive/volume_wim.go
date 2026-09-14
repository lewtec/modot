//go:build linux || windows

package archive

import (
	"io"

	"github.com/lewtec/lewkit/x/fs/wim"
)

// ExtractWIM unpacks one WIM image into destDir. image is 1-based.
// r must be an [io.ReaderAt]; otherwise [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
func ExtractWIM(r io.Reader, destDir string, image int) error {
	img, err := wim.Open(r, image)
	if err != nil {
		return err
	}
	defer img.Close()
	return ExtractFS(destDir, img)
}
