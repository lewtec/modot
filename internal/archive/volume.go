package archive

import (
	"io"

	"github.com/lewtec/lewkit/x/fs/udf"
)

// ExtractUDF unpacks a UDF volume (ISO) into destDir.
// r must be an [io.ReaderAt]; otherwise [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
func ExtractUDF(r io.Reader, destDir string) error {
	vol, err := udf.Open(r)
	if err != nil {
		return err
	}
	return ExtractFS(destDir, vol)
}
