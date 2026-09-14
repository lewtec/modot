package archive

import "strings"

// IsTarName reports a tar archive, including compressed wrappers
// (.tar.gz, .tgz, .tar.xz, .txz, .tar.zst, …).
func IsTarName(name string) bool {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".tar"),
		strings.HasSuffix(n, ".tgz"),
		strings.HasSuffix(n, ".tbz"),
		strings.HasSuffix(n, ".tbz2"),
		strings.HasSuffix(n, ".txz"):
		return true
	default:
		return strings.Contains(n, ".tar.")
	}
}

// IsZipName reports a .zip name.
func IsZipName(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".zip")
}

// IsSquashFSName reports a SquashFS image name.
func IsSquashFSName(name string) bool {
	n := strings.ToLower(name)
	return strings.HasSuffix(n, ".sfs") || strings.HasSuffix(n, ".squashfs")
}
