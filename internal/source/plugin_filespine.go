package source

import (
	"context"

	"github.com/lewtec/modot/internal/configcue"
)

// FileSpinePlugin lowers pipeline files into file profiles and
// emits dest files. Open on the dest FS is the combined file.
type FileSpinePlugin struct {
	cfg        *configcue.Config
	targetBase string
	modulesDir string
}

func NewFileSpinePlugin(cfg *configcue.Config, targetBase string) *FileSpinePlugin {
	return &FileSpinePlugin{cfg: cfg, targetBase: targetBase}
}

func (p *FileSpinePlugin) Name() string { return "file-spine" }

func (p *FileSpinePlugin) Process(ctx context.Context, files []File) ([]File, error) {
	tree, err := composeApply(ctx, destRequest{config: p.cfg, targetBase: p.targetBase, modulesDir: p.modulesDir, files: files})
	if err != nil {
		return nil, err
	}
	return tree.Files(), nil
}
