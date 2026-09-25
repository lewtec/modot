package types

import (
	"fmt"
	"os"
	"path/filepath"
)

// DaemonSocketPath is the unix socket used by the modot daemon and its clients.
func DaemonSocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(runtimeDir, "modot.sock")
}
