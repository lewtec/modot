package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/modot/internal/cmdarg"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	"github.com/lewtec/modot/internal/filespine"
	"github.com/lewtec/modot/internal/logging"
	"github.com/lewtec/modot/internal/module"
	"github.com/lewtec/modot/internal/placestep"
)

func init() {
	module.RegisterCoreModule(placeModule{})
}

type placeModule struct{}

func (placeModule) Ref() string { return "place" }

func (placeModule) Prepare(ctx context.Context, cfg map[string]any, resolver module.SourceRefResolver, modulesBaseDir string) error {
	if raw, ok := cfg["items"]; ok {
		if items, ok := raw.(map[string]any); ok {
			for dest, v := range items {
				if s, ok := v.(string); ok {
					resolved, did, err := resolver(ctx, s, modulesBaseDir)
					if err != nil {
						return fmt.Errorf("items[%q]: %w", dest, err)
					}
					if did {
						items[dest] = resolved
					}
				}
			}
		}
	}
	return nil
}

// placeConfig for core:place.
//
//	items: {
//	  ".grok/skills": "mySkills:."
//	}
//	ignore_missing: true
//	steps: {
//	  "10_require": { op: "require", patterns: { skill: "SKILL.md" } }
//	  "20_demote":  { op: "move", from: "SKILL.md", to: "entry.md" }
//	}
type placeConfig struct {
	Items         map[string]string    `json:"items"`
	IgnoreMissing bool                 `json:"ignore_missing"`
	Steps         map[string]placeStep `json:"steps"`
}

type placeStep struct {
	Op       string            `json:"op"`
	From     string            `json:"from"`
	To       string            `json:"to"`
	Patterns map[string]string `json:"patterns"`
}

func (step placeStep) asPlace() placestep.Step {
	return placestep.Step{Op: step.Op, From: step.From, To: step.To, Patterns: step.Patterns}
}

func (placeModule) Resolve(ctx context.Context, req module.ResolveRequest) (module.ResolveResult, error) {
	logger := logging.GetLogger(ctx)
	if !filespine.NamespaceVisible(req.Config.RuntimeMode(), filespine.ModeHome) {
		return module.ResolveResult{}, nil
	}

	cfg, err := module.DecodeConfig[placeConfig](req.ModuleConfig)
	if err != nil {
		return module.ResolveResult{}, fmt.Errorf("module %s: %w", req.ModuleName, err)
	}

	home := cmdarg.PrefixPath(ctx)
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return module.ResolveResult{}, err
		}
	}

	var (
		out      []module.ResolvedFile
		warnings []string
	)

	for dest, src := range cfg.Items {
		s := strings.TrimSpace(src)
		if s == "" {
			continue
		}

		srcPath := envdriver.ExpandPath(s)
		destClean := strings.Trim(dest, "/")

		st, err := os.Stat(srcPath)
		if err != nil {
			if cfg.IgnoreMissing && errors.Is(err, os.ErrNotExist) {
				logger.Info("place: skipping missing source", "module", req.ModuleName, "dest", dest, "source", srcPath)
				continue
			}
			return module.ResolveResult{}, fmt.Errorf("place source %q: %w", srcPath, err)
		}

		entries, err := collectPlaceEntries(srcPath, st)
		if err != nil {
			return module.ResolveResult{}, err
		}

		steps := make(map[string]placestep.Step, len(cfg.Steps))
		for name, step := range cfg.Steps {
			steps[name] = step.asPlace()
		}
		subject := fmt.Sprintf("place module %q item %q", req.ModuleName, destClean)
		entries, stepWarns, err := placestep.Apply(subject, steps, entries)
		if err != nil {
			return module.ResolveResult{}, err
		}
		for _, w := range stepWarns {
			logger.Warn(w)
			warnings = append(warnings, w)
		}

		for _, e := range entries {
			finalRel := e.Rel
			if destClean != "" && destClean != "." {
				finalRel = filepath.Join(destClean, filepath.FromSlash(e.Rel))
			} else {
				finalRel = filepath.FromSlash(e.Rel)
			}
			out = append(out, module.ResolvedFile{
				RelPath:    finalRel,
				TargetBase: home,
				Mode:       e.Mode,
				Info:       fmt.Sprintf("module:%s place (%s)", req.ModuleName, finalRel),
				AbsPath:    e.Abs,
				Symlink:    e.Symlink,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return module.ResolveResult{Files: out, Warnings: warnings}, nil
}

func collectPlaceEntries(srcPath string, st os.FileInfo) ([]placestep.Entry, error) {
	if !st.IsDir() {
		base := filepath.Base(srcPath)
		return []placestep.Entry{{
			Rel:     filepath.ToSlash(base),
			Abs:     srcPath,
			Mode:    st.Mode(),
			Symlink: st.Mode()&os.ModeSymlink != 0,
		}}, nil
	}

	var entries []placestep.Entry
	err := filepath.Walk(srcPath, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcPath, p)
		if err != nil {
			return err
		}
		entries = append(entries, placestep.Entry{
			Rel:     filepath.ToSlash(rel),
			Abs:     p,
			Mode:    info.Mode(),
			Symlink: info.Mode()&os.ModeSymlink != 0,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func cleanPlacePath(p string) (string, error) {
	return placestep.Clean(p)
}
