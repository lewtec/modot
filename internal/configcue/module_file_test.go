package configcue

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/fs/compose"
	_ "github.com/lucasew/workspaced/pkg/driver/env/native"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/pelletier/go-toml/v2"
)

func TestModuleFileLiftsIntoWorkspacedFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "modules", "greet")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`package module

module: {
	meta: {requires: [], recommends: []}
	config: {
		name: string | *"world"
	}
	file: home: "hello.json": {
		type: "json"
		values: {
			ok:   true
			name: workspaced.modules.greet.config.name
		}
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cuePath := filepath.Join(root, "workspaced.cue")
	if err := os.WriteFile(cuePath, []byte(`package workspaced
workspaced: {
	modules: greet: {
		input:  "self"
		path:   "modules/greet"
		enable: true
		config: {name: "ada"}
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := logging.NewWriterContext(t.Output())
	cfg, err := LoadFiles(ctx, []string{cuePath})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := cfg.FileProfiles()
	if err != nil {
		t.Fatal(err)
	}
	home := parsed["home"]
	if home == nil {
		t.Fatalf("profiles: %v", profileNames(parsed))
	}
	fsys, err := home.FS(nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := fs.ReadFile(fsys, "hello.json")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json %s: %v", body, err)
	}
	if got["ok"] != true {
		t.Fatalf("ok = %#v", got["ok"])
	}
	if got["name"] != "ada" {
		t.Fatalf("name = %#v", got["name"])
	}
}

func TestModuleFileUserOverlayWithoutType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "modules", "greet")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`package module

module: {
	meta: {requires: [], recommends: []}
	config: {}
	file: home: "hello.toml": {
		type: "toml"
		values: {
			onboarding: false
		}
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cuePath := filepath.Join(root, "workspaced.cue")
	if err := os.WriteFile(cuePath, []byte(`package workspaced
workspaced: {
	modules: greet: {
		input:  "self"
		path:   "modules/greet"
		enable: true
	}
	file: home: "hello.toml": values: {
		if workspaced.runtime.goos != "" {
			terminal: default_shell: "/opt/homebrew/bin/bash"
		}
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := logging.NewWriterContext(t.Output())
	cfg, err := LoadFiles(ctx, []string{cuePath})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := cfg.FileProfiles()
	if err != nil {
		t.Fatal(err)
	}
	home := parsed["home"]
	if home == nil {
		t.Fatalf("profiles: %v", profileNames(parsed))
	}
	fsys, err := home.FS(nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := fs.ReadFile(fsys, "hello.toml")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(body, &got); err != nil {
		t.Fatalf("toml %s: %v", body, err)
	}
	if got["onboarding"] != false {
		t.Fatalf("onboarding = %#v\n%s", got["onboarding"], body)
	}
	term, _ := got["terminal"].(map[string]any)
	if term["default_shell"] != "/opt/homebrew/bin/bash" {
		t.Fatalf("terminal = %#v\n%s", got["terminal"], body)
	}
	if strings.TrimSpace(string(body)) == "" {
		t.Fatal("empty toml")
	}
}

func profileNames(profiles map[string]*compose.Tree) []string {
	out := make([]string, 0, len(profiles))
	for name := range profiles {
		out = append(out, name)
	}
	return out
}
