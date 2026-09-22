package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/lucasew/workspaced/internal/module"
	"github.com/lucasew/workspaced/internal/modulecue"
	"github.com/lucasew/workspaced/pkg/filespine"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrModuleNotFound           = errors.New("workspace module not found")
	ErrStrictStructureViolation = errors.New("strict structure violation: file found in module root")
	ErrUnknownPreset            = errors.New("unknown preset")
	ErrMissingModuleCue         = errors.New("module is missing module.cue")
)

func init() {
	module.RegisterProvider(&Provider{})
}

type Provider struct{}

func (p *Provider) ID() string   { return "self" }
func (p *Provider) Name() string { return "Workspace Module" }

func resolvePresetBase(name, modulesBaseDir, prefix string) (string, error) {
	switch name {
	case "home":
		if prefix != "" && prefix != "~" {
			return prefix, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		return home, nil
	case "codebase":
		return filepath.Clean(filepath.Dir(modulesBaseDir)), nil
	case "system", "root":
		return filespine.ApplyDir(name, prefix), nil
	default:
		if !filespine.IsNamespace(name) {
			return "", fmt.Errorf("%w: %q", ErrUnknownPreset, name)
		}
		return filespine.ApplyDir(name, prefix), nil
	}
}

func (p *Provider) Resolve(ctx context.Context, req module.ResolveRequest) (module.ResolveResult, error) {
	workspaceRoot := filepath.Dir(req.ModulesBaseDir)
	modPath := req.Ref
	if !filepath.IsAbs(modPath) {
		modPath = filepath.Join(workspaceRoot, req.Ref)
	}
	if st, err := os.Stat(modPath); err != nil || !st.IsDir() {
		return module.ResolveResult{}, fmt.Errorf("%w: %q at %s", ErrModuleNotFound, req.Ref, modPath)
	}

	if err := validateConfig(req.Ref, modPath, req.ModuleConfig, req.Config.Raw()); err != nil {
		return module.ResolveResult{}, err
	}

	entries, err := os.ReadDir(modPath)
	if err != nil {
		return module.ResolveResult{}, err
	}

	var out []module.ResolvedFile
	for _, preset := range entries {
		if !preset.IsDir() {
			name := strings.TrimSpace(preset.Name())
			if name == "README.md" || strings.HasSuffix(name, ".cue") {
				continue
			}
			return module.ResolveResult{}, fmt.Errorf("%w: %q in module %q", ErrStrictStructureViolation, name, req.Ref)
		}
		presetName := preset.Name()
		targetBase, err := resolvePresetBase(presetName, req.ModulesBaseDir, req.Prefix)
		if err != nil {
			return module.ResolveResult{}, fmt.Errorf("%w in module %q", err, req.Ref)
		}
		mode := filespine.ModeHome
		if req.Config != nil {
			mode = req.Config.RuntimeMode()
		}
		if !filespine.NamespaceVisible(mode, presetName) {
			continue
		}

		presetPath := filepath.Join(modPath, presetName)
		err = filepath.Walk(presetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(presetPath, path)
			if err != nil {
				return err
			}
			isSymlink := info.Mode()&os.ModeSymlink != 0
			out = append(out, module.ResolvedFile{
				RelPath:    rel,
				TargetBase: targetBase,
				Mode:       info.Mode(),
				Info:       fmt.Sprintf("module:%s (%s/%s)", req.Ref, presetName, rel),
				AbsPath:    path,
				Symlink:    isSymlink,
			})
			return nil
		})
		if err != nil {
			return module.ResolveResult{}, err
		}
	}

	return module.ResolveResult{Files: out}, nil
}

func validateConfig(modName string, modPath string, modCfg map[string]any, root map[string]any) error {
	if !modulecue.Exists(modPath) {
		return fmt.Errorf("%w: %q", ErrMissingModuleCue, modName)
	}
	return modulecue.ValidateConfigWithRoot(modPath, modCfg, root)
}
