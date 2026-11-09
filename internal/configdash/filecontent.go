package configdash

import (
	"errors"
	"fmt"
)

// ErrFileNotFound is returned when the tree has no file at the requested path.
var ErrFileNotFound = errors.New("no such file in this scope")

// FindFile picks one file out of a report by its root and its path relative to
// that root. Both are needed because one scope can place files under several
// roots, and the same relative path can exist under more than one.
func FindFile(rep FilesReport, filePath, root string) (MaterialFile, error) {
	if rep.Err != nil {
		return MaterialFile{}, rep.Err
	}
	for _, file := range rep.Files {
		if file.Path == filePath && file.Root == root {
			return file, nil
		}
	}

	return MaterialFile{}, fmt.Errorf("%w: %s under %s", ErrFileNotFound, filePath, root)
}
