package tool

import (
	"path/filepath"

	kittool "github.com/lewtec/lewkit/x/tool"

	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
)

func GetToolsDir() (string, error) {
	return workspacedShareDir("tools")
}

func GetShimsDir() (string, error) {
	return workspacedShareDir("shims")
}

func workspacedShareDir(leaf string) (string, error) {
	home, err := envdriver.ResolveHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "workspaced", leaf), nil
}

// FindBinary searches for cmdName under baseDir (bin/ and the directory root).
func FindBinary(baseDir, cmdName string) string {
	return kittool.FindBinary(baseDir, cmdName)
}

// BinaryCandidates lists the paths FindBinary checks, in order.
func BinaryCandidates(baseDir, cmdName string) []string {
	return kittool.BinaryCandidates(baseDir, cmdName)
}
