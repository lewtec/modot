package filespine

import (
	"path/filepath"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// IsTemplatePath is true when any suffix is .tmpl, and for a .d.tmpl fragment.
// file.tmpl.sh is a template. Compose squash only looks at the last suffix.
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
