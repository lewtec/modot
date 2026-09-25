package source

import (
	"context"
	"io/fs"

	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/template"
	"github.com/lewtec/modot/internal/logging"
)

// Tree is dest files after profiles compose.
// Apply writes Files. It does not rebuild the tree.
type Tree struct {
	dest       fs.FS
	targetBase string
	files      []File
	Warnings   []string
}

// NewApplyTree is a dest-less Tree for callers that already have apply files.
func NewApplyTree(files []File, warnings ...string) *Tree {
	return &Tree{files: files, Warnings: warnings}
}

// Dest is the encoded dest tree. Open returns the combined file.
// Paths are relative. When two profiles share a path, Open returns the first.
func (t *Tree) Dest() fs.FS {
	if t == nil {
		return nil
	}
	return t.dest
}

// Files is the apply view of Dest (paths relative to TargetBase).
func (t *Tree) Files() []File {
	if t == nil {
		return nil
	}
	return t.files
}

// TargetBase is the default physical root for dest paths.
func (t *Tree) TargetBase() string {
	if t == nil {
		return ""
	}
	return t.targetBase
}

// Builder discovers source files, then composes cue + static + templates.
type Builder struct {
	Config     *configcue.Config
	TargetBase string
	Providers  []Plugin
}

// Tree runs discovery, renders templates, then composes one tree per profile.
func (b Builder) Tree(ctx context.Context) (*Tree, error) {
	var warnings []string
	ctx = WithWarningSink(ctx, &warnings)

	files, err := NewPipeline(b.Providers...).Run(ctx, nil)
	if err != nil {
		return nil, err
	}
	static, tmpl := splitTemplateFiles(files)
	rendered, err := NewTemplateExpanderPlugin(template.NewEngine(ctx), b.Config).Process(ctx, tmpl)
	if err != nil {
		return nil, err
	}
	discovered := make([]File, 0, len(static)+len(rendered))
	discovered = append(discovered, static...)
	discovered = append(discovered, rendered...)
	tree, err := composeApply(ctx, destRequest{config: b.Config, targetBase: b.TargetBase, files: discovered})
	if err != nil {
		return nil, err
	}
	tree.Warnings = warnings
	logging.GetLogger(ctx).Debug("dest composed", "files", len(tree.files))
	return tree, nil
}
