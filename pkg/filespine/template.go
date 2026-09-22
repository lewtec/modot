package filespine

import lewpath "github.com/lewtec/lewkit/x/path"

// IsTemplatePath is true when any suffix is .tmpl, and for a .d.tmpl fragment.
// file.tmpl.sh is a template. Compose squash only looks at the last suffix.
func IsTemplatePath(rel string) bool {
	for current := lewpath.New(rel); ; current = current.Parent() {
		for _, suffix := range current.Suffixes() {
			if suffix == ".tmpl" {
				return true
			}
		}
		if current.Parent() == current {
			return false
		}
	}
}
