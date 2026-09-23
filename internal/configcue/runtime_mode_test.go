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
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "load home")
	require.Equal(t, filespine.ModeHome, home.RuntimeMode())
	homeFiles, err := home.FileProfiles()
	require.NoError(t, err)
	require.NotContains(t, homeFiles, "codebase", "home mode emitted codebase")
	homeFS, err := homeFiles["home"].FS(nil)
	require.NoError(t, err)
	_, err = fs.Stat(homeFS, ".codex/config.toml")
	require.NoError(t, err, "home profile")

	code, err := loadFilesMode(ctx, cuePath, filespine.ModeCodebase)
	require.NoError(t, err, "load codebase")
	require.Equal(t, filespine.ModeCodebase, code.RuntimeMode())
	codeFiles, err := code.FileProfiles()
	require.NoError(t, err)
	require.NotContains(t, codeFiles, "home", "codebase mode emitted home")
	codeFS, err := codeFiles["codebase"].FS(nil)
	require.NoError(t, err)
	_, err = fs.Stat(codeFS, ".gitignore")
	require.NoError(t, err, "codebase profile")
}

func TestFlatFileKeyRejected(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "workspaced.cue"), `package workspaced
workspaced: file: ".bashrc": {type: "lines", values: {"00": "umask 022"}}
`)
	ctx := logging.NewWriterContext(t.Output())
	_, err := loadFilesMode(ctx, filepath.Join(root, "workspaced.cue"), filespine.ModeHome)
	require.Error(t, err, "expected schema error")
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
	require.NoError(t, err, "load")
	_, err = cfg.FileProfiles()
	require.Error(t, err, "expected ref error")
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
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
