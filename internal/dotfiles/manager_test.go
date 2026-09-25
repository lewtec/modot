package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/deployer"
	"github.com/lewtec/modot/internal/source"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestApplyPersistsDropOfGitignoredStateOnIdle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ignoredRel := filepath.Join(".grok", "old.md")
	keptRel := "README.md"
	ignoredAbs := filepath.Join(root, ignoredRel)
	keptAbs := filepath.Join(root, keptRel)
	content := []byte("same\n")
	for _, p := range []string{ignoredAbs, keptAbs} {
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, content, 0o644))
	}
	st, err := os.Stat(keptAbs)
	require.NoError(t, err)
	mode := st.Mode()

	tree := source.NewApplyTree([]source.File{
		&source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    ignoredRel,
				TargetBaseDir: root,
				FileMode:      mode,
				Info:          "module:place (old.md)",
				FileType:      source.TypeStatic,
			},
			Content: content,
		},
		&source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    keptRel,
				TargetBaseDir: root,
				FileMode:      mode,
				Info:          "module:place (README.md)",
				FileType:      source.TypeStatic,
			},
			Content: content,
		},
	})

	statePath := filepath.Join(root, ".modot", "state.json")
	store, err := deployer.NewFileStateStore(statePath, root)
	require.NoError(t, err)
	require.NoError(t, store.Save(&deployer.State{Files: map[string]deployer.ManagedInfo{
		ignoredAbs: {SourceInfo: "module:place (old.md)"},
		keptAbs:    {SourceInfo: "module:place (README.md)"},
	}}))

	mgr, err := NewManager(Config{
		Tree:       tree,
		StateStore: store,
		Ignore:     func(target string) bool { return target == ignoredAbs },
	})
	require.NoError(t, err)

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	_ = g
	result, err := mgr.Apply(ctx, ApplyOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.StateDropped)
	require.Equal(t, 0, result.FilesCreated+result.FilesUpdated+result.FilesDeleted,
		"want idle apply, got create=%d update=%d delete=%d",
		result.FilesCreated, result.FilesUpdated, result.FilesDeleted)

	loaded, err := store.Load()
	require.NoError(t, err)
	require.NotContains(t, loaded.Files, ignoredAbs, "gitignored key still in state after idle apply")
	require.Contains(t, loaded.Files, keptAbs, "tracked key missing after idle apply")
	_, err = os.Stat(ignoredAbs)
	require.NoError(t, err, "ignored file should stay on disk")
}
