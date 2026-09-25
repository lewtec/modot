package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/filespine"
	"github.com/lewtec/modot/internal/git"
	"github.com/lewtec/modot/internal/module"
	"github.com/lewtec/modot/internal/modulecue"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrModuleNotFound           = errors.New("workspace module not found")
	ErrStrictStructureViolation = errors.New("strict structure violation: file found in module root")
	ErrUnknownPreset            = errors.New("unknown preset")
	ErrMissingModuleCue         = errors.New("module is missing modot.cue")
)

func init() {
	module.RegisterProvider(&Provider{})
}

type Provider struct{}

func (p *Provider) ID() string   { return "self" }
func (p *Provider) Name() string { return "Workspace Module" }

func resolvePresetBase(ctx context.Context, name, modulesBaseDir string) (string, error) {
	root := cmdarg.PrefixPath(ctx)
	switch name {
	case "home":
		if root != "" {
			return root, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		return home, nil
	case "codebase":
		if root != "" {
			return root, nil
		}
		return filepath.Clean(filepath.Dir(modulesBaseDir)), nil
	default:
		if filespine.SystemRel(name) == "" && name != "root" {
			return "", fmt.Errorf("%w: %q", ErrUnknownPreset, name)
		}
		if root == "" {
			root = "/"
		}
		return root, nil
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

	if !modulecue.Exists(modPath) {
		return module.ResolveResult{}, fmt.Errorf("%w: %q", ErrMissingModuleCue, req.Ref)
	}
	templates, err := moduleContributesTemplates(ctx, modPath)
	if err != nil {
		return module.ResolveResult{}, err
	}
	if !templates {
		return module.ResolveResult{}, nil
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
		targetBase, err := resolvePresetBase(ctx, presetName, req.ModulesBaseDir)
		if err != nil {
			return module.ResolveResult{}, fmt.Errorf("%w in module %q", err, req.Ref)
		}
		mode := filespine.ModeHome
		if req.Config != nil {
			mode = req.Config.RuntimeMode()
		}
		if !filespine.PresetVisible(mode, presetName) {
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
			if prefix := filespine.SystemRel(presetName); prefix != "" && prefix != "." {
				rel = filepath.Join(prefix, rel)
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

func moduleContributesTemplates(ctx context.Context, modPath string) (bool, error) {
	root, err := git.GetRoot(ctx, modPath)
	if err != nil || strings.TrimSpace(root) == "" {
		return false, nil
	}
	absMod, err := filepath.Abs(modPath)
	if err != nil {
		return false, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	return filepath.Clean(absMod) != filepath.Clean(absRoot), nil
}
