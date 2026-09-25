package filespine

import (
	"errors"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
)

var (
	errDirectoryNotAbsolute = errors.New("directory is not absolute")
)

// HostPath appends a relative name to an OS directory.
// The relative name stays a Path. The OS directory is the string passed to path.Open.
func HostPath(root string, rel lewpath.Path) string {
	if rel == lewpath.New(".") {
		if root == "" {
			return "."
		}
		return root
	}
	tail := strings.Join(rel.Parts(), "/")
	switch {
	case root == "" || root == ".":
		return tail
	case root == "/":
		return "/" + tail
	default:
		return strings.TrimRight(root, "/") + "/" + tail
	}
}

// OpenDir opens an absolute directory.
// The opened root's name is the host path passed to path.Open.
func OpenDir(dir lewpath.Path) (*lewpath.Root, error) {
	if !dir.IsAbs() {
		return nil, errDirectoryNotAbsolute
	}
	return lewpath.Open(dir.String())
}
