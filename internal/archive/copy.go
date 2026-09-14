package archive

import (
	"errors"
	"fmt"
	"io"
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

// CopyFS writes every regular file and directory from src into dst.
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
		info, err := d.Info()
		if err != nil {
			return err
		}
		f, err := src.Open(name)
		if err != nil {
			return err
		}
		writeErr := WriteMemberFS(dst, p, info.Mode(), f)
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	})
}

// WriteMemberFS creates name in dst and copies r into it.
// On failure after create, name is removed.
func WriteMemberFS(dst *xpath.Root, name xpath.Path, mode fs.FileMode, r io.Reader) error {
	if !name.Valid() || name.IsAbs() || name.String() == "." {
		return fmt.Errorf("%w: %s", ErrIllegalPath, name)
	}
	if mode == 0 {
		mode = 0o644
	}
	if parent := name.Parent(); parent.String() != "." {
		if err := parent.MkdirAll(dst, 0o755); err != nil {
			return err
		}
	}
	f, err := name.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	wc, ok := f.(io.WriteCloser)
	if !ok {
		return errors.Join(fmt.Errorf("%w: %s", xpath.ErrReadOnly, name), f.Close())
	}
	_, copyErr := io.Copy(wc, r)
	closeErr := wc.Close()
	if copyErr != nil {
		return errors.Join(copyErr, removeMember(dst, name))
	}
	if closeErr != nil {
		return errors.Join(closeErr, removeMember(dst, name))
	}
	if mode&0o111 != 0 {
		if err := name.Chmod(dst, mode); err != nil {
			return errors.Join(fmt.Errorf("set permissions: %w", err), removeMember(dst, name))
		}
	}
	return nil
}

func removeMember(dst *xpath.Root, name xpath.Path) error {
	err := name.Remove(dst)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
