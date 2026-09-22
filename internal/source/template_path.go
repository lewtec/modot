package source

import lewpath "github.com/lewtec/lewkit/x/path"

// isTemplatePath reports a file the template expander renders.
// A .tmpl suffix anywhere in the file name counts, so file.tmpl.sh is a template.
// A .d.tmpl directory is not itself the template; compose squash merges those fragments.
func isTemplatePath(rel string) bool {
	for _, suffix := range lewpath.New(rel).Suffixes() {
		if suffix == ".tmpl" {
			return true
		}
	}
	return false
}

// stripTemplate removes one .tmpl suffix from the file name.
// file.tmpl becomes file. file.tmpl.sh becomes file.sh.
func stripTemplate(rel string) string {
	name := lewpath.New(rel)
	if name.Suffix() == ".tmpl" {
		return name.WithSuffix("").String()
	}
	stem := lewpath.New(name.Stem())
	if stem.Suffix() != ".tmpl" {
		return name.String()
	}
	return name.WithStem(stem.WithSuffix("").String()).String()
}
