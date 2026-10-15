package env

import (
	"context"
	"os"
	"path/filepath"

	"github.com/lewtec/modot/internal/api"
)

// DotfilesCandidates is the list of paths to search for dotfiles root.
// Paths are checked in order, first match wins.
// Supports ~ for home directory and $VAR for environment variables.
var DotfilesCandidates = []string{
	"/workspaces/.codespaces/.persistedshare/dotfiles", // GitHub Codespaces
	"~/.dotfiles",    // User dotfiles
	"/etc/.dotfiles", // System-wide dotfiles
}

// FindDotfilesRoot walks DotfilesCandidates using home for ~ expansion.
func FindDotfilesRoot(home string) (string, error) {
	for _, path := range DotfilesCandidates {
		expanded := ExpandPathIn(path, home)
		if info, err := os.Stat(expanded); err == nil && info.IsDir() {
			return expanded, nil
		}
	}
	return "", api.ErrDotfilesRootNotFound
}

// EnsureUnderHome joins home/rel, creates the directory, and returns it.
func EnsureUnderHome(home, rel string) (string, error) {
	path := filepath.Join(home, rel)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// Hostname is os.Hostname with a stable error path for drivers.
func Hostname(ctx context.Context) (string, error) {
	return os.Hostname()
}
