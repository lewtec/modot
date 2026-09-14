package archive

import (
	"fmt"
	"io/fs"
	"os"

	xpath "github.com/lewtec/lewkit/x/path"
)

// ExtractFS copies src into destDir through a jailed [xpath.Root].
func ExtractFS(destDir string, src fs.FS) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	root, err := xpath.Open(destDir)
	if err != nil {
		return err
	}
	defer root.Close()
	return CopyFS(root, src)
}

// CopyFS writes src into dst using [xpath.Path] I/O.
// Names that are not valid [io/fs] paths are [ErrIllegalPath].
func CopyFS(dst *xpath.Root, src fs.FS) error {
	return xpath.New(".").WalkDir(src, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		p := xpath.New(name)
		if name != "." && (!p.Valid() || p.IsAbs() || p.String() == ".") {
			return fmt.Errorf("%w: %s", ErrIllegalPath, name)
		}
		if name == "." || d.IsDir() {
			if name == "." {
				return nil
			}
			return p.MkdirAll(dst, 0o755)
		}
		if parent := p.Parent(); parent.String() != "." {
			if err := parent.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		}
		st, err := d.Info()
		if err != nil {
			return err
		}
		if st.Mode()&fs.ModeSymlink != 0 {
			target, err := p.ReadLink(src)
			if err != nil {
				return err
			}
			return p.Symlink(dst, target)
		}
		data, err := p.ReadFile(src)
		if err != nil {
			return err
		}
		mode := st.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		return p.WriteFile(dst, data, mode)
	})
}
