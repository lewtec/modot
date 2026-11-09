package configdash_test

import (
	"testing"

	"github.com/lewtec/modot/internal/configdash"
	_ "github.com/lewtec/modot/internal/driver/prelude"
	"github.com/stretchr/testify/require"
)

// fileNode walks a name path down a tree, failing when a step is missing.
func fileNode(t *testing.T, nodes []configdash.FileNode, names ...string) configdash.FileNode {
	t.Helper()
	var found configdash.FileNode
	current := nodes
	for _, name := range names {
		found = configdash.FileNode{}
		for _, node := range current {
			if node.Name == name {
				found = node

				break
			}
		}
		require.NotEmpty(t, found.Name, "no node named %q", name)
		current = found.Children
	}

	return found
}

// The page shows a tree, not a table, so the tree has to nest: a directory
// names itself once and the files under it carry only their own names.
func TestFileTreeNestsDirectories(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "LoadFiles")

	tree := files.FileTree()
	require.Len(t, tree, 1, "one apply root")

	// The fixture's two files sit at the root, so the tree starts with the
	// root directory and lists them as leaves.
	top := tree[0]
	require.True(t, top.IsDir(), "the root is a directory")
	require.Contains(t, childNames(top), "from-module.txt")
	require.Contains(t, childNames(top), "from-tree.txt")

	module := fileNode(t, top.Children, "from-module.txt")
	require.False(t, module.IsDir())
	require.NotNil(t, module.File)
	// A leaf is named by its base name; the page reads the directory from
	// the parent, so the path is not repeated in every row.
	require.Equal(t, "from-module.txt", module.Name)
	require.Equal(t, "from-module.txt", module.File.Path)
}

// A tree's count is how many files the branch holds, so a collapsed directory
// still says how much is under it.
func TestFileTreeCountsChildren(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "LoadFiles")

	total := 0
	for _, node := range files.FileTree() {
		total += node.Files
	}
	require.Equal(t, files.MaterialCount(), total)

	for _, node := range files.FileTree() {
		for _, child := range node.Children {
			require.Positive(t, child.Files, "leaf %q counts itself", child.Name)
		}
	}
}

// childNames lists a node's children in the order the page renders them.
func childNames(node configdash.FileNode) []string {
	names := make([]string, 0, len(node.Children))
	for _, child := range node.Children {
		names = append(names, child.Name)
	}

	return names
}

// Contents come from the tree, read in the same request that built it: the
// bytes are what apply writes, so a rendered template shows its output.
func TestLoadFilesIncludesContent(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "LoadFiles")

	byPath := map[string]configdash.MaterialFile{}
	for _, file := range files.Files {
		byPath[file.Path] = file
	}

	module := byPath["from-module.txt"]
	require.True(t, module.Text(), "a text file has its bytes")
	require.Contains(t, module.Content.Text, "from the module")
	require.Equal(t, 1, module.Content.Lines)
	require.False(t, module.Content.Truncated)
	require.Empty(t, module.ContentNote())
}

// A path the tree does not hold is not an error page: it names the file that
// was asked for and says the scope does not place it.
func TestFindFile(t *testing.T) {
	t.Parallel()
	root := filesFixtureRoot(t)

	files, err := configdash.LoadFiles(t.Context(), configdash.ScopeCodebase, root)
	require.NoError(t, err, "LoadFiles")

	target := ""
	for _, file := range files.Files {
		if file.Path == "from-module.txt" {
			target = file.Root
		}
	}
	require.NotEmpty(t, target)

	found, findErr := configdash.FindFile(files, "from-module.txt", target)
	require.NoError(t, findErr, "FindFile")
	require.Equal(t, "from-module.txt", found.Path)

	_, missing := configdash.FindFile(files, "nope.txt", target)
	require.ErrorIs(t, missing, configdash.ErrFileNotFound)
}
