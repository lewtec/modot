package archive

import (
	"fmt"
	"os"
	"path/filepath"
)

// StripTopLevelDir hoists children when destPath contains exactly one directory.
func StripTopLevelDir(destPath string) error {
	entries, err := os.ReadDir(destPath)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return nil
	}

	singleDir := filepath.Join(destPath, entries[0].Name())
	tempDir := destPath + ".strip-tmp"
	if err := os.Rename(singleDir, tempDir); err != nil {
		return err
	}

	tempEntries, err := os.ReadDir(tempDir)
	if err != nil {
		if restoreErr := os.Rename(tempDir, singleDir); restoreErr != nil {
			return fmt.Errorf("%w; restore failed: %w", err, restoreErr)
		}
		return err
	}

	for _, entry := range tempEntries {
		oldPath := filepath.Join(tempDir, entry.Name())
		newPath := filepath.Join(destPath, entry.Name())
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}
	return os.Remove(tempDir)
}
