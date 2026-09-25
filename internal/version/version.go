package version

import (
	"runtime/debug"
	"strings"

	"github.com/lewtec/lewkit/x/release"
)

var version = "dev"

// Version returns the modot version.
// It defaults to "dev" when ldflags injection is not provided.
func Version() string {
	v := strings.TrimSpace(version)
	if v == "" {
		return "dev"
	}
	return v
}

// BuildID returns the build ID from buildinfo
func BuildID() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	// Try to get vcs.revision for commit hash
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			if len(setting.Value) > 8 {
				return setting.Value[:8] // short hash
			}
			return setting.Value
		}
	}

	return "dev"
}

// Platform is [release.Platform]: GOOS-GOARCH[-microarch].
func Platform() string {
	return release.Platform()
}

// GetBuildID returns a build identifier combining version and commit hash.
// No platform: keep it filename-safe (shell-init cache keys).
func GetBuildID() string {
	return Version() + "-" + BuildID()
}

// VersionString is the full --version line body: "<buildID> <platform>".
func VersionString() string {
	return GetBuildID() + " " + Platform()
}
