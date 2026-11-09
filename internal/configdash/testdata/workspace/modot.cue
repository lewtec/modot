package modot

modules: {
	theme: {input: "self:modules/theme", enable: true}
}

options: theme: {
	accent: "blue"
}

file: home: {
	".config/theme.toml": {
		type:   "toml"
		values: accent: options.theme.accent
	}
}
