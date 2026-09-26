package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/modot/internal/configcue"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	"github.com/lewtec/modot/internal/modfile"
	"github.com/lewtec/modot/internal/placestep"
)

// mountExpand is one profile path whose sources are walked and stepped.
type mountExpand struct {
	cfg        *configcue.Config
	modulesDir string
	dest       string
	spec       configcue.Mount
	targetBase string
}

// expandMount walks each source, runs place steps, and returns files rooted at targetBase.
// dest is the profile-relative prefix. A file source with no steps lands on dest itself.
func expandMount(ctx context.Context, job mountExpand) ([]File, error) {
	job.dest = strings.Trim(filepath.ToSlash(job.dest), "/")
	var out []File
	for _, src := range job.spec.Srcs {
		files, err := job.one(ctx, src)
		if err != nil {
			return nil, err
		}
		out = append(out, files...)
	}
	return out, nil
}

func (job mountExpand) one(ctx context.Context, src string) ([]File, error) {
	resolved, err := job.resolve(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("mount %s: %w", job.dest, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("mount %s: %w", job.dest, err)
	}
	entries, err := mountEntries(resolved, info)
	if err != nil {
		return nil, fmt.Errorf("mount %s: %w", job.dest, err)
	}
	subject := fmt.Sprintf("mount %q", job.dest)
	entries, warnings, err := placestep.Apply(subject, job.spec.Steps, entries)
	if err != nil {
		return nil, fmt.Errorf("mount %s: %w", job.dest, err)
	}
	for _, warning := range warnings {
		AppendWarning(ctx, warning)
	}
	out := make([]File, 0, len(entries))
	fileAtDest := !info.IsDir() && len(job.spec.Steps) == 0 && len(entries) == 1
	for _, entry := range entries {
		rel := entry.Rel
		if fileAtDest {
			rel = job.dest
			if rel == "" || rel == "." {
				rel = entry.Rel
			}
		} else if job.dest != "" && job.dest != "." {
			rel = job.dest + "/" + entry.Rel
		}
		out = append(out, mountedFile(rel, entry, job.targetBase))
	}
	return out, nil
}

func mountEntries(src string, info os.FileInfo) ([]placestep.Entry, error) {
	if !info.IsDir() {
		return []placestep.Entry{{
			Rel:     filepath.ToSlash(filepath.Base(src)),
			Abs:     src,
			Mode:    info.Mode(),
			Symlink: info.Mode()&os.ModeSymlink != 0,
		}}, nil
	}
	var entries []placestep.Entry
	err := filepath.Walk(src, func(path string, walkInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if walkInfo.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		entries = append(entries, placestep.Entry{
			Rel:     filepath.ToSlash(rel),
			Abs:     path,
			Mode:    walkInfo.Mode(),
			Symlink: walkInfo.Mode()&os.ModeSymlink != 0,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func mountedFile(rel string, entry placestep.Entry, targetBase string) File {
	kind := TypeStatic
	if entry.Symlink {
		kind = TypeSymlink
	}
	return &StaticFile{
		BasicFile: BasicFile{
			RelPathStr:    rel,
			TargetBaseDir: targetBase,
			FileMode:      entry.Mode,
			Info:          "mount:" + rel,
			FileType:      kind,
		},
		AbsPath: entry.Abs,
	}
}

func (job mountExpand) resolve(ctx context.Context, src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", errEmptyMountSrc
	}
	if job.cfg != nil {
		mod, err := modfile.ModFileFromConfig(job.cfg)
		if err != nil {
			return "", err
		}
		resolved, ok, err := mod.TryResolveSourceRefToPath(ctx, src, job.modulesDir)
		if err != nil {
			return "", err
		}
		if ok {
			src = resolved
		}
	}
	return envdriver.ExpandPath(src), nil
}

var errEmptyMountSrc = errors.New("empty src")
