package deployer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGitWorkTree(t *testing.T, gitignore string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
	if gitignore != "" {
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644))
	}
	return root
}

func TestGitignoreUntrackedWalksUpToGitRoot(t *testing.T) {
	t.Parallel()
	gitRoot := writeGitWorkTree(t, ".grok/\n")
	nested := filepath.Join(gitRoot, "apps", "svc")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	ignore := GitignoreUntracked(nested)
	require.NotNil(t, ignore, "expected ignore fn from parent git root")
	require.True(t, ignore(filepath.Join(nested, ".grok", "skill.md")), "nested .grok path should match parent .gitignore")
	require.False(t, ignore(filepath.Join(nested, "main.go")), "nested tracked path should not be ignored")
}

func TestGitignoreUntrackedNilWithoutGit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".grok/\n"), 0o644))
	require.Nil(t, GitignoreUntracked(root), "expected nil ignore without .git")
}

func TestGitignoreUntrackedMatchesRepoIgnore(t *testing.T) {
	t.Parallel()
	root := writeGitWorkTree(t, ".grok/\n*.local\n!keep.local\n")
	ignore := GitignoreUntracked(root)
	require.NotNil(t, ignore, "expected ignore fn")

	tests := []struct {
		rel  string
		want bool
	}{
		{".grok/skills/foo.md", true},
		{"keep.local", false},
		{"foo.local", true},
		{"README.md", false},
	}
	for _, tt := range tests {
		got := ignore(filepath.Join(root, filepath.FromSlash(tt.rel)))
		assert.Equal(t, tt.want, got, "ignore(%q)", tt.rel)
	}
	assert.False(t, ignore(filepath.Join(t.TempDir(), "outside")), "path outside root should not be ignored")
}

func TestGitignoreUntrackedNestedAndExclude(t *testing.T) {
	t.Parallel()
	root := writeGitWorkTree(t, "")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("scratch/\n"), 0o644))
	nested := filepath.Join(root, "pkg")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, ".gitignore"), []byte("gen/\n"), 0o644))

	ignore := GitignoreUntracked(root)
	require.NotNil(t, ignore, "expected ignore fn")
	assert.True(t, ignore(filepath.Join(root, "scratch", "a.txt")), "info/exclude scratch/ should ignore")
	assert.True(t, ignore(filepath.Join(root, "pkg", "gen", "out.go")), "nested pkg/.gitignore gen/ should ignore")
	assert.False(t, ignore(filepath.Join(root, "pkg", "main.go")), "pkg/main.go should not be ignored")
}

func TestDropIgnored(t *testing.T) {
	t.Parallel()
	root := writeGitWorkTree(t, ".grok/\n")
	ignore := GitignoreUntracked(root)
	tracked := filepath.Join(root, "README.md")
	untracked := filepath.Join(root, ".grok", "a.md")
	state := &State{Files: map[string]ManagedInfo{
		tracked:   {SourceInfo: "a"},
		untracked: {SourceInfo: "b"},
	}}
	n := DropIgnored(state, ignore)
	require.Equal(t, 1, n)
	require.Contains(t, state.Files, tracked, "tracked key dropped")
	require.NotContains(t, state.Files, untracked, "ignored key still in state")
}
