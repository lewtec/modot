package archive

import (
	lewpath "github.com/lewtec/lewkit/x/path"
)

// StripTopLevelDir hoists children when destPath contains exactly one directory.
func StripTopLevelDir(destPath string) error {
	root, err := lewpath.Open(destPath)
	if err != nil {
		return err
	}
	defer root.Close()
	ents, err := lewpath.New(".").ReadDir(root)
	if err != nil {
		return err
	}
	if len(ents) != 1 || !ents[0].IsDir() {
		return nil
	}
	child := lewpath.New(ents[0].Name())
	kids, err := child.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range kids {
		from := child.Join(e.Name())
		if err := from.Rename(root, lewpath.New(e.Name())); err != nil {
			return err
		}
	}
	return child.RemoveAll(root)
}
