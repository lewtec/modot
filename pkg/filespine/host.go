package filespine

import (
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
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
// The path is slash algebra from an OS path; the opened root's name is the host path.
func OpenDir(dir lewpath.Path) (*lewpath.Root, error) {
	slash, err := lewpath.Open("/")
	if err != nil {
		return nil, err
	}
	rel, err := dir.Rel(lewpath.New("/"))
	if err != nil {
		slash.Close()
		return nil, err
	}
	if rel == lewpath.New(".") {
		return slash, nil
	}
	opened, err := rel.OpenRoot(slash)
	slash.Close()
	if err != nil {
		return nil, err
	}
	return opened, nil
}
