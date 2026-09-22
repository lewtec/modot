package source

import (
	"context"

	"github.com/lucasew/workspaced/internal/configcue"
)

// FileSpinePlugin lowers pipeline files into workspaced.file profiles and
// emits dest files. Open on the dest FS is the combined file.
type FileSpinePlugin struct {
	cfg        *configcue.Config
	targetBase string
}

func NewFileSpinePlugin(cfg *configcue.Config, targetBase string) *FileSpinePlugin {
	return &FileSpinePlugin{cfg: cfg, targetBase: targetBase}
}

func (p *FileSpinePlugin) Name() string { return "file-spine" }

func (p *FileSpinePlugin) Process(ctx context.Context, files []File) ([]File, error) {
	_ = ctx
	tree, err := composeApply(p.cfg, p.targetBase, files)
	if err != nil {
		return nil, err
	}
	return tree.Files(), nil
}
