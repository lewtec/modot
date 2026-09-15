package filespine

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// StaticDir walks Root and emits ref slots for non-template files.
type StaticDir struct {
	Label string
	Root  string
}

func (d StaticDir) Name() string {
	if d.Label != "" {
		return d.Label
	}
	return "static:" + d.Root
}

func (d StaticDir) Provide(ctx context.Context) (Patch, error) {
	_ = ctx
	root, err := lewpath.Open(d.Root)
	if err != nil {
		return Patch{}, err
	}
	defer root.Close()
	var slots []Contribution
	err = lewpath.New(".").WalkDir(root, func(name string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		p := lewpath.New(name)
		if info.IsDir() {
			if strings.HasSuffix(p.Name(), ".d.tmpl") {
				return fs.SkipDir
			}
			return nil
		}
		rel := p.String()
		if rel == "." || IsTemplatePath(rel) {
			return nil
		}
		st, err := info.Info()
		if err != nil {
			return err
		}
		mode := st.Mode()
		if mode == 0 {
			mode = 0o644
		}
		slots = append(slots, Contribution{
			Path:    rel,
			Type:    TypeRef,
			Key:     "src",
			Slot:    Slot{Kind: KindRef, Ref: filepath.Join(d.Root, filepath.FromSlash(rel))},
			Mode:    mode.Perm(),
			Info:    fmt.Sprintf("static:%s", rel),
			Symlink: mode&os.ModeSymlink != 0,
		})
		return nil
	})
	if err != nil {
		return Patch{}, err
	}
	return Patch{Slots: slots}, nil
}

// IsTemplatePath is true for .tmpl files and .d.tmpl fragments.
func IsTemplatePath(rel string) bool {
	p := lewpath.New(filepath.ToSlash(rel))
	s := p.String()
	if strings.Contains(s, ".d.tmpl/") || strings.HasSuffix(s, ".d.tmpl") {
		return true
	}
	for _, suf := range p.Suffixes() {
		if suf == ".tmpl" {
			return true
		}
	}
	return false
}
