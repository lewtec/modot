package configcue

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeOptionsModule(t *testing.T, root, name, options, extra string) {
	t.Helper()
	modDir := filepath.Join(root, "modules", name)
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "modot.cue"), []byte(`package module

options: {
`+options+`
}
`+extra), 0o644))
}

func writeModotCue(t *testing.T, root, config string) string {
	t.Helper()
	cuePath := filepath.Join(root, "modot.cue")
	require.NoError(t, os.WriteFile(cuePath, []byte(config), 0o644))
	return cuePath
}

func evaluateOptionsConfig(t *testing.T, root, config string) map[string]any {
	t.Helper()
	writeModotCue(t, root, config)
	res, err := Evaluate(t.Context(), DiscoverOptions{Cwd: root, Mode: "home"})
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(res.JSON, &got))
	return got
}

func optionsOf(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	options, ok := got["options"].(map[string]any)
	require.True(t, ok, "options: %v", got["options"])
	return options
}

func TestModuleOptionsBecomeConfigSchema(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "greet", `name: string | *"world"`, `
file: home: "hello.json": {
	type: "json"
	values: {
		name: options.greet.name
	}
}
`)
	cuePath := writeModotCue(t, root, `package modot
modules: greet: {
	input:  "self"
	path:   "modules/greet"
	enable: true
}
`)

	cfg, err := LoadFiles(t.Context(), []string{cuePath})
	require.NoError(t, err)
	parsed, err := cfg.FileProfiles()
	require.NoError(t, err)
	home := parsed["home"]
	require.NotNil(t, home, "profiles: %v", profileNames(parsed))
	fsys, err := home.FS(nil)
	require.NoError(t, err)
	body, err := fs.ReadFile(fsys, "hello.json")
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(body, &got), "json %s", body)
	require.Equal(t, "world", got["name"])
}

// Options are one shared namespace: a module that declares options.base16 and
// another that declares options.theme both resolve, and neither needs a
// module prefix beyond its own name under options.
func TestModuleOptionsShareOneNamespace(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "base16", `
	mode: *"dark" | "light"
	palette: base00: *"#1d2021" | string
`, "")
	writeOptionsModule(t, root, "theme", `
	mode: *"dark" | "light"
	palette: options.base16.palette
`, "")

	got := evaluateOptionsConfig(t, root, `package modot
modules: {
	base16: {input: "self", path: "modules/base16", enable: true}
	theme: {input: "self", path: "modules/theme", enable: true}
}
options: theme: mode: "light"
`)
	options := optionsOf(t, got)
	require.Equal(t, "light", options["theme"].(map[string]any)["mode"])
	palette := options["theme"].(map[string]any)["palette"].(map[string]any)
	require.Equal(t, "#1d2021", palette["base00"])
}

// A hyphen in a module name is a hard error: options live at options.<name>,
// so the author must pick a name CUE can select with dot notation.
func TestModuleNameWithHyphenFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "foo-bar", `name: *"foo" | string`, "")
	cuePath := writeModotCue(t, root, `package modot
modules: "foo-bar": {
	input:  "self"
	path:   "modules/foo-bar"
	enable: true
}
`)

	_, err := LoadFiles(t.Context(), []string{cuePath})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a CUE identifier")
}

func TestModuleOptionsSetFromUserConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "greet", `name: string | *"world"`, "")

	got := evaluateOptionsConfig(t, root, `package modot
options: greet: name: "from-options"
modules: greet: {
	input:  "self"
	path:   "modules/greet"
	enable: true
}
`)
	require.Equal(t, "from-options", optionsOf(t, got)["greet"].(map[string]any)["name"])
}

// modules.<name>.config no longer exists: the module entry holds only
// enable/input/path/from/version and options carry the configuration.
func TestModuleEntryHasNoConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "greet", `name: *"world" | string`, "")

	got := evaluateOptionsConfig(t, root, `package modot
modules: greet: {
	input:  "self"
	path:   "modules/greet"
	enable: true
}
`)
	modules := got["modules"].(map[string]any)
	entry := modules["greet"].(map[string]any)
	require.NotContains(t, entry, "config", "module entry: %v", entry)
	require.Equal(t, "world", optionsOf(t, got)["greet"].(map[string]any)["name"])
}

func TestModuleConfigWriteIsRejected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "greet", `name: string | *"world"`, "")
	cuePath := writeModotCue(t, root, `package modot
modules: greet: {
	input:  "self"
	path:   "modules/greet"
	enable: true
	config: name: "nope"
}
`)

	_, err := LoadFiles(t.Context(), []string{cuePath})
	require.Error(t, err)
}

// Two modules that declare the same option path with different types must
// fail loudly instead of silently picking one.
func TestModuleOptionsConflictFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "one", `shared: string`, "")
	writeOptionsModule(t, root, "two", `shared: int`, "")
	cuePath := writeModotCue(t, root, `package modot
modules: {
	one: {input: "self", path: "modules/one", enable: true}
	two: {input: "self", path: "modules/two", enable: true}
}
`)

	_, err := LoadFiles(t.Context(), []string{cuePath})
	require.Error(t, err)
}

func TestModuleOptionsRejectBadConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeOptionsModule(t, root, "greet", `name: string`, "")
	cuePath := writeModotCue(t, root, `package modot
options: greet: name: 42
modules: greet: {
	input:  "self"
	path:   "modules/greet"
	enable: true
}
`)

	_, err := LoadFiles(t.Context(), []string{cuePath})
	require.Error(t, err)
}
