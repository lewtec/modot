package source

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasew/workspaced/internal/configcue"
	_ "github.com/lucasew/workspaced/pkg/driver/env/native"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestBuilderTreeRendersTemplateAndStatic(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	src := t.TempDir()
	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "hello.txt.tmpl"), []byte("hi {{ .runtime.goos }}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "plain.txt"), []byte("static\n"), 0o644))
	scanner, err := NewScannerPlugin(ScannerConfig{Name: "src", BaseDir: src, TargetBase: dest})
	require.NoError(t, err)
	tree, err := Builder{Config: &configcue.Config{}, TargetBase: dest, Providers: []Plugin{scanner}}.Tree(ctx)
	require.NoError(t, err)
	require.NotNil(t, tree.Dest(), "Dest is nil")
	got, err := fs.ReadFile(tree.Dest(), "plain.txt")
	require.NoError(t, err)
	require.Equal(t, "static\n", string(got))
	hello, err := fs.ReadFile(tree.Dest(), "hello.txt")
	require.NoError(t, err)
	require.NotEmpty(t, hello, "hello.txt empty")
	require.Len(t, tree.Files(), 2)
}

func TestBuilderTreeMergesCueLines(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	src := t.TempDir()
	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, ".bashrc.d.tmpl"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, ".bashrc.d.tmpl", "10-mod.sh"), []byte("from-mod"), 0o644))
	cuePath := filepath.Join(t.TempDir(), "workspaced.cue")
	require.NoError(t, os.WriteFile(cuePath, []byte(`package workspaced
workspaced: {
	file: home: ".bashrc": {
		type: "lines"
		values: {"00-cue": "from-cue"}
	}
}
`), 0o644))
	cfg, err := configcue.LoadFiles(ctx, []string{cuePath})
	require.NoError(t, err)
	scanner, err := NewScannerPlugin(ScannerConfig{Name: "src", BaseDir: src, TargetBase: dest})
	require.NoError(t, err)
	tree, err := Builder{Config: cfg, TargetBase: dest, Providers: []Plugin{scanner}}.Tree(ctx)
	require.NoError(t, err)
	got, err := fs.ReadFile(tree.Dest(), ".bashrc")
	require.NoError(t, err)
	require.Equal(t, "from-cue\nfrom-mod", string(got))
}

func TestNewApplyTreeHasNoDest(t *testing.T) {
	t.Parallel()
	tree := NewApplyTree([]File{&BufferFile{
		BasicFile: BasicFile{RelPathStr: "a", TargetBaseDir: "/tmp"},
		Content:   []byte("x"),
	}})
	require.Nil(t, tree.Dest())
	require.Len(t, tree.Files(), 1)
}
