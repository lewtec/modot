package configdash

import (
	"path"
	"sort"
	"strings"
)

// FileNode is one entry of a scope's file tree: a directory, or a file apply
// places. The tree exists so a page names a directory once and lists what is
// under it, instead of repeating the directory on every row.
type FileNode struct {
	// Name is the node's name: a directory's base name, or a file's.
	Name string
	// Dir is the path relative to the apply root, "/" separated; "" is the
	// root itself.
	Dir string
	// Root is the apply root the node hangs from, e.g. ~ or /.
	Root string
	// File is the file at this node, nil for a directory.
	File *MaterialFile
	// Children are the files and directories under this node, directories
	// first and then by name, so a tree reads the way a file browser shows it.
	Children []FileNode
	// Files is how many files are under this node, itself included.
	Files int
}

// IsDir reports whether the node is a directory.
func (n FileNode) IsDir() bool { return n.File == nil }

// FilesReport.FileTree groups the files the way a reader navigates them: one
// node per directory rather than one row per file repeating its path.
func (rep FilesReport) FileTree() []FileNode {
	return buildFileTree(rep.Files)
}

// dirNode is a directory while the tree is built, before folding turns it into
// a FileNode.
type dirNode struct {
	// name is the directory's own base name.
	name string
	// dir is the path relative to root.
	dir string
	// root is the apply root.
	root string
	// files are the files landing directly in this directory.
	files []MaterialFile
	// childDir are the subdirectories by base name.
	childDir map[string]*dirNode
}

// buildFileTree nests files under their directories. A directory with one
// child directory and no files of its own is folded into that child, so a deep
// module tree shows the part that differs instead of a column of one-child
// directories.
func buildFileTree(files []MaterialFile) []FileNode {
	roots := map[string]*dirNode{}
	order := []string{}
	nodeFor := func(root string) *dirNode {
		node, ok := roots[root]
		if !ok {
			node = &dirNode{name: root, dir: "", root: root, files: nil, childDir: map[string]*dirNode{}}
			roots[root] = node
			order = append(order, root)
		}

		return node
	}
	for i := range files {
		file := files[i]
		node := nodeFor(file.Root)
		if file.Dir == "" {
			node.files = append(node.files, file)

			continue
		}
		segments := strings.Split(file.Dir, "/")
		for _, segment := range segments {
			child, ok := node.childDir[segment]
			if !ok {
				dir := segment
				if node.dir != "" {
					dir = node.dir + "/" + segment
				}
				child = &dirNode{name: segment, dir: dir, root: node.root, files: nil, childDir: map[string]*dirNode{}}
				node.childDir[segment] = child
			}
			node = child
		}
		node.files = append(node.files, file)
	}

	trees := make([]FileNode, 0, len(order))
	for _, root := range order {
		trees = append(trees, flattenDir(roots[root]))
	}

	return trees
}

// flattenDir turns a directory into a node: its files become file nodes and
// its subdirectories recurse. A single-child chain with no files of its own
// folds, so the node name reads as "a/b/c" and the page does not spend a line
// on each step.
func flattenDir(node *dirNode) FileNode {
	name := node.name
	for len(node.files) == 0 && len(node.childDir) == 1 {
		var only *dirNode
		for _, child := range node.childDir {
			only = child
		}
		node = only
		name += "/" + only.name
	}
	out := FileNode{Name: name, Dir: node.dir, Root: node.root, File: nil, Children: nil, Files: 0}
	out.Files = len(node.files)
	files := make([]MaterialFile, len(node.files))
	copy(files, node.files)
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	for i := range files {
		file := files[i]
		child := FileNode{
			Name:     file.Name,
			Dir:      file.Dir,
			Root:     file.Root,
			File:     &file,
			Children: nil,
			Files:    1,
		}
		out.Children = append(out.Children, child)
	}
	dirs := make([]string, 0, len(node.childDir))
	for childName := range node.childDir {
		dirs = append(dirs, childName)
	}
	sort.Strings(dirs)
	for _, childName := range dirs {
		child := flattenDir(node.childDir[childName])
		out.Files += child.Files
		out.Children = append(out.Children, child)
	}

	return out
}

// Path is the node's path relative to its apply root, or its label when it is
// the root itself.
func (n FileNode) Path() string {
	if n.File != nil {
		return n.File.Path
	}
	if n.Dir == "" {
		return n.Root
	}

	return n.Dir
}

// Target is where a file lands; for a directory it is the directory itself.
func (n FileNode) Target() string {
	if n.File != nil {
		return n.File.Target
	}
	if n.Dir == "" {
		return n.Root
	}

	return path.Join(n.Root, n.Dir)
}

// DirCount is how many directories the tree has, the apply roots included, so
// the summary can say how wide the tree is without keeping a second grouping
// of the same files.
func (rep FilesReport) DirCount() int {
	var count func(nodes []FileNode) int
	count = func(nodes []FileNode) int {
		total := 0
		for _, node := range nodes {
			if node.IsDir() {
				total++
				total += count(node.Children)
			}
		}

		return total
	}

	return count(rep.FileTree())
}
