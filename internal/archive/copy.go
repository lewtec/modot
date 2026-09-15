package archive

import (
	"context"
	"io/fs"
	"os"

	lewfs "github.com/lewtec/lewkit/x/fs"
	xpath "github.com/lewtec/lewkit/x/path"
)

// CopyTo writes files into destDir through a jailed [xpath.Root].
func CopyTo(ctx context.Context, destDir string, files lewfs.Files) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	dest, err := xpath.Open(destDir)
	if err != nil {
		return err
	}
	defer dest.Close()
	return lewfs.Copy(ctx, dest, files)
}

// ExtractFS copies src into destDir via [lewfs.Walk] and [lewfs.Copy].
func ExtractFS(ctx context.Context, destDir string, src fs.FS) error {
	return CopyTo(ctx, destDir, lewfs.Walk(src, nil))
}
