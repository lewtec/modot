package deployer

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	"github.com/stretchr/testify/require"
)

func TestMkdirAbs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	require.NoError(t, mkdirAbs(lewpath.New(dir)))
	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	require.NoError(t, mkdirAbs(lewpath.New("/")))
	require.EqualError(t, mkdirAbs(lewpath.New("relative")), "state directory is not absolute")
}

func TestFileStateStoreRelativeToRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(root, 0o755))
	statePath := filepath.Join(dir, "state.json")

	store, err := NewFileStateStore(statePath, root)
	require.NoError(t, err, "NewFileStateStore")

	absA := filepath.Join(root, "a", "file.txt")
	absB := filepath.Join(root, "b.txt")
	in := &State{Files: map[string]ManagedInfo{
		absA: {SourceInfo: "mod:a"},
		absB: {SourceInfo: "mod:b"},
	}}
	require.NoError(t, store.Save(in), "Save")

	raw, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var disk State
	require.NoError(t, json.Unmarshal(raw, &disk))
	require.Contains(t, disk.Files, filepath.Join("a", "file.txt"), "expected relative key a/file.txt in disk state, got %#v", disk.Files)
	require.Contains(t, disk.Files, "b.txt", "expected relative key b.txt in disk state, got %#v", disk.Files)
	for k := range disk.Files {
		require.False(t, filepath.IsAbs(k), "disk key should be relative, got %q", k)
	}

	loaded, err := store.Load()
	require.NoError(t, err, "Load")
	info, ok := loaded.Files[absA]
	require.True(t, ok, "load absA: got %#v", loaded.Files)
	require.Equal(t, "mod:a", info.SourceInfo)
	info, ok = loaded.Files[absB]
	require.True(t, ok, "load absB: got %#v", loaded.Files)
	require.Equal(t, "mod:b", info.SourceInfo)
}

func TestFileStateStoreSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(root, 0o755))
	statePath := filepath.Join(dir, "state.json")
	// Pre-existing state that must remain readable if replace is atomic.
	old := &State{Files: map[string]ManagedInfo{
		filepath.Join(root, "old.txt"): {SourceInfo: "old"},
	}}
	store, err := NewFileStateStore(statePath, root)
	require.NoError(t, err)
	require.NoError(t, store.Save(old))

	next := &State{Files: map[string]ManagedInfo{
		filepath.Join(root, "new.txt"): {SourceInfo: "new"},
	}}
	require.NoError(t, store.Save(next), "Save")

	_, err = os.Stat(statePath + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "temp file should be gone after successful Save")

	raw, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var disk State
	require.NoError(t, json.Unmarshal(raw, &disk), "final state must be valid JSON: %s", raw)
	require.Contains(t, disk.Files, "new.txt", "expected new.txt after save, got %#v", disk.Files)
	require.NotContains(t, disk.Files, "old.txt", "old.txt should have been replaced, got %#v", disk.Files)
}

func TestFileStateStoreMigratesAbsoluteKeys(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(root, 0o755))
	statePath := filepath.Join(dir, "state.json")
	abs := filepath.Join(root, "legacy.txt")

	// Legacy on-disk format: absolute keys.
	legacy := &State{Files: map[string]ManagedInfo{
		abs: {SourceInfo: "old"},
	}}
	data, err := json.MarshalIndent(legacy, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, data, 0o644))

	store, err := NewFileStateStore(statePath, root)
	require.NoError(t, err)
	loaded, err := store.Load()
	require.NoError(t, err)
	require.Contains(t, loaded.Files, abs, "expected absolute key after load, got %#v", loaded.Files)

	// Re-save should rewrite as relative.
	require.NoError(t, store.Save(loaded))
	raw, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var disk State
	require.NoError(t, json.Unmarshal(raw, &disk))
	require.Contains(t, disk.Files, "legacy.txt", "expected relativized key after save, got %#v", disk.Files)
}

func TestRelToRootAndAbsFromRoot(t *testing.T) {
	root := "/home/user"
	require.Equal(t, ".config/foo", RelToRoot("/home/user/.config/foo", root))
	require.Equal(t, "/home/user/.config/foo", AbsFromRoot(".config/foo", root))
	// Outside root stays absolute.
	require.Equal(t, "/other/x", RelToRoot("/other/x", root))
}

// A literal "$" in a managed filename must survive state round-tripping.
// os.ExpandEnv would treat "$inspect" as an unset variable and drop it,
// turning ".../$inspect.md" into ".../.md" and making every plan re-create the
// file and delete a phantom empty-named sibling.
func TestAbsFromRootKeepsLiteralDollar(t *testing.T) {
	root := "/home/user"
	in := ".agents/skills/svelte/$inspect.md"
	want := "/home/user/.agents/skills/svelte/$inspect.md"
	require.Equal(t, want, AbsFromRoot(in, root))

	t.Setenv("inspect", "EXPANDED")
	require.Equal(t, want, AbsFromRoot(in, root), "env var must not be substituted in a state key")
}

func TestExpandTildeInLeavesDollarAlone(t *testing.T) {
	require.Equal(t, "/data/home/.config/a/$inspect.md", envdriver.ExpandTildeIn("~/.config/a/$inspect.md", "/data/home"))
	require.Equal(t, "rel/$inspect.md", envdriver.ExpandTildeIn("rel/$inspect.md", "/data/home"))
	require.Equal(t, "/abs/$inspect.md", envdriver.ExpandTildeIn("/abs/$inspect.md", ""))
}

func TestPrettyPathUsesRelToRoot(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	got := PrettyPath(filepath.Join(home, ".config", "x"))
	require.Equal(t, "~/.config/x", got)
}
