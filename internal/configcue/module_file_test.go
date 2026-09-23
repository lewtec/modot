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
	"github.com/stretchr/testify/require"
)

func TestModuleFileLiftsIntoWorkspacedFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "modules", "greet")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`package module

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
`), 0o644))
	cuePath := filepath.Join(root, "workspaced.cue")
	require.NoError(t, os.WriteFile(cuePath, []byte(`package workspaced
workspaced: {
	modules: greet: {
		input:  "self"
		path:   "modules/greet"
		enable: true
		config: {name: "ada"}
	}
}
`), 0o644))
	ctx := logging.NewWriterContext(t.Output())
	cfg, err := LoadFiles(ctx, []string{cuePath})
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
	require.Equal(t, true, got["ok"])
	require.Equal(t, "ada", got["name"])
}

func TestModuleFileUserOverlayWithoutType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "modules", "greet")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`package module

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
`), 0o644))
	cuePath := filepath.Join(root, "workspaced.cue")
	require.NoError(t, os.WriteFile(cuePath, []byte(`package workspaced
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
`), 0o644))
	ctx := logging.NewWriterContext(t.Output())
	cfg, err := LoadFiles(ctx, []string{cuePath})
	require.NoError(t, err)
	parsed, err := cfg.FileProfiles()
	require.NoError(t, err)
	home := parsed["home"]
	require.NotNil(t, home, "profiles: %v", profileNames(parsed))
	fsys, err := home.FS(nil)
	require.NoError(t, err)
	body, err := fs.ReadFile(fsys, "hello.toml")
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, toml.Unmarshal(body, &got), "toml %s", body)
	require.Equal(t, false, got["onboarding"], "%s", body)
	term, _ := got["terminal"].(map[string]any)
	require.Equal(t, "/opt/homebrew/bin/bash", term["default_shell"], "terminal = %#v\n%s", got["terminal"], body)
	require.NotEmpty(t, strings.TrimSpace(string(body)), "empty toml")
}

func profileNames(profiles map[string]*compose.Tree) []string {
	out := make([]string, 0, len(profiles))
	for name := range profiles {
		out = append(out, name)
	}
	return out
}
