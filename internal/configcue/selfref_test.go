package configcue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A disabled module's file block must not be evaluated, and must not be able to
// reference options it would have declared.
func TestDisabledModuleFilesAreDropped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "base16", `
	mode: *"dark" | "light"
	palette: base00: *"#1d2021" | string
`, "")
	writeOptionsModule(t, root, "base16_alacritty", `
	palette: options.base16.palette
`, `
file: home: ".config/alacritty/colors.toml": {
	type: "toml"
	values: colors: primary: background: options.base16_alacritty.palette.base00
}
`)

	t.Run("enabled", func(t *testing.T) {
		got := evaluateOptionsConfig(t, root, `package modot
modules: {
	base16: {input: "self", path: "modules/base16", enable: true}
	base16_alacritty: {input: "self", path: "modules/base16_alacritty", enable: true}
}
`)
		require.Contains(t, got["options"].(map[string]any), "base16_alacritty")
		home := got["file"].(map[string]any)["home"].(map[string]any)
		require.Contains(t, home, ".config/alacritty/colors.toml")
	})

	t.Run("disabled", func(t *testing.T) {
		got := evaluateOptionsConfig(t, root, `package modot
modules: {
	base16: {input: "self", path: "modules/base16", enable: true}
	base16_alacritty: {input: "self", path: "modules/base16_alacritty", enable: false}
}
`)
		files, _ := got["file"].(map[string]any)
		home, _ := files["home"].(map[string]any)
		require.NotContains(t, home, ".config/alacritty/colors.toml")
	})
}

// A module reading another module's options must resolve regardless of the
// alphabetical order the loader uses for module names.
func TestOptionsResolveAcrossModuleOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "aaa_reader", `palette: options.zzz_source.palette`, `
file: home: "read.json": {
	type: "json"
	values: bg: options.aaa_reader.palette.base00
}
`)
	writeOptionsModule(t, root, "zzz_source", `palette: base00: *"#abcdef" | string`, "")

	got := evaluateOptionsConfig(t, root, `package modot
modules: {
	aaa_reader: {input: "self", path: "modules/aaa_reader", enable: true}
	zzz_source: {input: "self", path: "modules/zzz_source", enable: true}
}
`)
	reader := got["options"].(map[string]any)["aaa_reader"].(map[string]any)
	require.Equal(t, "#abcdef", reader["palette"].(map[string]any)["base00"])
	home := got["file"].(map[string]any)["home"].(map[string]any)
	require.Equal(t, "#abcdef", home["read.json"].(map[string]any)["values"].(map[string]any)["bg"])
}
