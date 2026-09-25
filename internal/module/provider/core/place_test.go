package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/modot/internal/cmdarg"
	"github.com/lewtec/modot/internal/module"
	"github.com/lewtec/modot/internal/logging"
)

func TestPlaceResolveIgnoreMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	existing := filepath.Join(root, "exists.txt")
	require.NoError(t, os.WriteFile(existing, []byte("hi"), 0o644))
	missing := filepath.Join(root, "nope.txt")

	ctx := logging.NewWriterContext(t.Output())
	m := placeModule{}

	t.Run("default fails on missing", func(t *testing.T) {
		t.Parallel()
		_, err := m.Resolve(ctx, module.ResolveRequest{
			ModuleName: "test-place",
			ModuleConfig: map[string]any{
				"items": map[string]any{
					"out": missing,
				},
			},
		})
		require.Error(t, err, "expected error for missing source")
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("ignore_missing skips missing keeps present", func(t *testing.T) {
		t.Parallel()
		out, err := m.Resolve(ctx, module.ResolveRequest{
			ModuleName: "test-place",
			ModuleConfig: map[string]any{
				"ignore_missing": true,
				"items": map[string]any{
					"out-missing": missing,
					"out-ok":      existing,
				},
			},
		})
		require.NoError(t, err, "Resolve")
		require.Len(t, out.Files, 1)
		require.Equal(t, "out-ok/exists.txt", out.Files[0].RelPath)
		require.Equal(t, existing, out.Files[0].AbsPath)
	})

	t.Run("ignore_missing all missing yields empty", func(t *testing.T) {
		t.Parallel()
		out, err := m.Resolve(ctx, module.ResolveRequest{
			ModuleName: "test-place",
			ModuleConfig: map[string]any{
				"ignore_missing": true,
				"items": map[string]any{
					"out": missing,
				},
			},
		})
		require.NoError(t, err, "Resolve")
		require.Empty(t, out.Files)
	})
}

func TestPlacePrefixWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(src, []byte("a"), 0o644))
	root := t.TempDir()
	ctx := cmdarg.WithPrefix(logging.NewWriterContext(t.Output()), root)
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "test-place",
		ModuleConfig: map[string]any{
			"items": map[string]any{
				".": src,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, out.Files, 1)
	require.Equal(t, root, out.Files[0].TargetBase)
}

func TestPlaceResolveDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "tree")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "test-place",
		ModuleConfig: map[string]any{
			"items": map[string]any{
				".config/app": dir,
			},
		},
	})
	require.NoError(t, err, "Resolve")
	require.Len(t, out.Files, 2)
	// sorted by RelPath
	require.Equal(t, ".config/app/a.txt", out.Files[0].RelPath)
	require.Equal(t, ".config/app/sub/b.txt", out.Files[1].RelPath)
}

func TestPlaceStepsMoveAndRequire(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "skill")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	skillPath := filepath.Join(pkg, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte("# skill"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "notes.md"), []byte("n"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "best_practices",
		ModuleConfig: map[string]any{
			"items": map[string]any{
				"skills/best-practices/references/go": pkg,
			},
			"steps": map[string]any{
				"10_require": map[string]any{
					"op": "require",
					"patterns": map[string]any{
						"skill": "SKILL.md",
					},
				},
				"20_demote": map[string]any{
					"op":   "move",
					"from": "SKILL.md",
					"to":   "entry.md",
				},
			},
		},
	})
	require.NoError(t, err, "Resolve")
	rels := map[string]string{}
	for _, f := range out.Files {
		rels[f.RelPath] = f.AbsPath
	}
	wantEntry := "skills/best-practices/references/go/entry.md"
	require.Equal(t, skillPath, rels[wantEntry], "files=%v", rels)
	_, ok := rels["skills/best-practices/references/go/SKILL.md"]
	require.False(t, ok, "SKILL.md should have been moved")
	_, ok = rels["skills/best-practices/references/go/notes.md"]
	require.True(t, ok, "notes.md missing")
}

func TestPlaceRequireFailsWhenMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "skill")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "README.md"), []byte("x"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	_, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "best_practices",
		ModuleConfig: map[string]any{
			"items": map[string]any{"out": pkg},
			"steps": map[string]any{
				"10_require": map[string]any{
					"op":       "require",
					"patterns": map[string]any{"skill": "SKILL.md"},
				},
			},
		},
	})
	require.Error(t, err, "expected require failure")
	require.ErrorIs(t, err, errPlaceRequireNoMatch)
}

func TestPlaceRequireNegation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "skill")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "SKILL.md"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "entry.md"), []byte("y"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	_, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "best_practices",
		ModuleConfig: map[string]any{
			"items": map[string]any{"out": pkg},
			"steps": map[string]any{
				"10_require": map[string]any{
					"op": "require",
					"patterns": map[string]any{
						"skill":    "SKILL.md",
						"no_entry": "!entry.md",
					},
				},
			},
		},
	})
	require.Error(t, err, "expected negation failure")
	require.ErrorIs(t, err, errPlaceRequireMustNot)
}

func TestPlaceMoveMissingWarns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "skill")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "a.md"), []byte("a"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "m",
		ModuleConfig: map[string]any{
			"items": map[string]any{"out": pkg},
			"steps": map[string]any{
				"20_demote": map[string]any{
					"op":   "move",
					"from": "SKILL.md",
					"to":   "entry.md",
				},
			},
		},
	})
	require.NoError(t, err, "Resolve")
	require.Len(t, out.Warnings, 1)
	require.Contains(t, out.Warnings[0], "missing")
	require.Len(t, out.Files, 1)
	require.True(t, strings.HasSuffix(out.Files[0].RelPath, "a.md"))
}

func TestPlaceMoveOverwriteWarns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "skill")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	skill := filepath.Join(pkg, "SKILL.md")
	entry := filepath.Join(pkg, "entry.md")
	require.NoError(t, os.WriteFile(skill, []byte("skill"), 0o644))
	require.NoError(t, os.WriteFile(entry, []byte("old"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "m",
		ModuleConfig: map[string]any{
			"items": map[string]any{"out": pkg},
			"steps": map[string]any{
				"20_demote": map[string]any{
					"op":   "move",
					"from": "SKILL.md",
					"to":   "entry.md",
				},
			},
		},
	})
	require.NoError(t, err, "Resolve")
	require.Len(t, out.Warnings, 1)
	require.Contains(t, out.Warnings[0], "overwrites")
	require.Len(t, out.Files, 1)
	require.Equal(t, skill, out.Files[0].AbsPath)
}

func TestPlaceMoveDirPrefix(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pkg := filepath.Join(root, "skill")
	require.NoError(t, os.MkdirAll(filepath.Join(pkg, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "docs", "a.md"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "keep.md"), []byte("k"), 0o644))

	ctx := logging.NewWriterContext(t.Output())
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "m",
		ModuleConfig: map[string]any{
			"items": map[string]any{"out": pkg},
			"steps": map[string]any{
				"10_mv": map[string]any{
					"op":   "move",
					"from": "docs",
					"to":   "reference",
				},
			},
		},
	})
	require.NoError(t, err, "Resolve")
	rels := map[string]bool{}
	for _, f := range out.Files {
		rels[f.RelPath] = true
	}
	require.True(t, rels["out/reference/a.md"], "expected dir prefix move: %v", rels)
	require.False(t, rels["out/docs/a.md"], "docs should be gone: %v", rels)
	require.True(t, rels["out/keep.md"], "keep missing: %v", rels)
}

func TestPlaceUnknownOpSkipped(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	a := filepath.Join(root, "a")
	require.NoError(t, os.WriteFile(a, []byte("a"), 0o644))
	ctx := logging.NewWriterContext(t.Output())
	out, err := placeModule{}.Resolve(ctx, module.ResolveRequest{
		ModuleName: "m",
		ModuleConfig: map[string]any{
			"items": map[string]any{"out": root},
			"steps": map[string]any{
				"x": map[string]any{"op": "reject"},
			},
		},
	})
	require.NoError(t, err, "Resolve")
	require.Len(t, out.Files, 1)
}

func TestCleanPlacePath(t *testing.T) {
	t.Parallel()
	_, err := cleanPlacePath("../x")
	require.ErrorIs(t, err, errPlacePathEscape)
	_, err = cleanPlacePath("")
	require.ErrorIs(t, err, errPlaceEmptyPath)
	got, err := cleanPlacePath("foo/bar")
	require.NoError(t, err)
	require.Equal(t, "foo/bar", got)
}
