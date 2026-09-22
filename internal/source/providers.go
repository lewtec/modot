package source

import "github.com/lucasew/workspaced/pkg/filespine"

func splitTemplateFiles(files []File) (static, tmpl []File) {
	for _, f := range files {
		if filespine.IsTemplatePath(f.RelPath()) {
			tmpl = append(tmpl, f)
			continue
		}
		static = append(static, f)
	}
	return static, tmpl
}
