package configcue

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lucasew/workspaced/pkg/driver/env/native"
	"github.com/lucasew/workspaced/pkg/filespine"
	"github.com/lucasew/workspaced/pkg/logging"
)

func TestRuntimeModeAndFileProfiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workspaced.cue"), `package workspaced
workspaced: file: {
	home: {
		".codex/config.toml": {type: "toml", values: {model: "x"}}
	}
	codebase: {
		".gitignore": {type: "text", values: {content: "bin/"}}
	}
}
`)

	ctx := logging.NewWriterContext(t.Output())
	cuePath := filepath.Join(root, "workspaced.cue")
	home, err := loadFilesMode(ctx, cuePath, filespine.ModeHome)
	if err != nil {
		t.Fatalf("load home: %v", err)
	}
	if got := home.RuntimeMode(); got != filespine.ModeHome {
		t.Fatalf("home mode=%q", got)
	}
	homeFiles, err := home.FileProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := homeFiles["codebase"]; ok {
		t.Fatal("home mode emitted codebase")
	}
	homeFS, err := homeFiles["home"].FS(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(homeFS, ".codex/config.toml"); err != nil {
		t.Fatalf("home profile: %v", err)
	}

	code, err := loadFilesMode(ctx, cuePath, filespine.ModeCodebase)
	if err != nil {
		t.Fatalf("load codebase: %v", err)
	}
	if got := code.RuntimeMode(); got != filespine.ModeCodebase {
		t.Fatalf("codebase mode=%q", got)
	}
	codeFiles, err := code.FileProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := codeFiles["home"]; ok {
		t.Fatal("codebase mode emitted home")
	}
	codeFS, err := codeFiles["codebase"].FS(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(codeFS, ".gitignore"); err != nil {
		t.Fatalf("codebase profile: %v", err)
	}
}

func TestFlatFileKeyRejected(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workspaced.cue"), `package workspaced
workspaced: file: ".bashrc": {type: "lines", values: {"00": "umask 022"}}
`)
	ctx := logging.NewWriterContext(t.Output())
	_, err := loadFilesMode(ctx, filepath.Join(root, "workspaced.cue"), filespine.ModeHome)
	if err == nil {
		t.Fatal("expected schema error")
	}
}

func TestAbsoluteRefRejected(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workspaced.cue"), `package workspaced
workspaced: file: home: blob: {
	type: "ref"
	values: src: {kind: "ref", ref: "/tmp/x"}
}
`)
	ctx := logging.NewWriterContext(t.Output())
	cfg, err := loadFilesMode(ctx, filepath.Join(root, "workspaced.cue"), filespine.ModeHome)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err = cfg.FileProfiles()
	if err == nil {
		t.Fatal("expected ref error")
	}
}

func loadFilesMode(ctx context.Context, path, mode string) (*Config, error) {
	v, err := buildWorkspacedValue(ctx, []string{path}, nil, DiscoverOptions{Mode: mode})
	if err != nil {
		return nil, err
	}
	data, err := marshalWorkspacedValue(ctx, v, []string{path}, nil)
	if err != nil {
		return nil, err
	}
	cfg, err := decodeConfig(data)
	if err != nil {
		return nil, err
	}
	cfg.cueVal = v
	return cfg, nil
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
