package source

func splitTemplateFiles(files []File) (static, tmpl []File) {
	for _, f := range files {
		if isTemplatePath(f.RelPath()) {
			tmpl = append(tmpl, f)
			continue
		}
		static = append(static, f)
	}
	return static, tmpl
}
