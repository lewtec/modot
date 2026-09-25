package deployer

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/modot/internal/cmdctx"
	"github.com/lewtec/modot/internal/source"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestPlannerDetectsCommentOnlyContentChange(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rg")
	require.NoError(t, os.WriteFile(target, []byte("#!/usr/bin/env bash\n# locked: old\nexec true\n"), 0o755), "write target")

	desired := []DesiredState{{
		File: &source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    "rg",
				TargetBaseDir: dir,
				FileMode:      0o755,
				Info:          "source:config-tree (.local/bin/_index.tmpl) (multi:rg)",
				FileType:      source.TypeMultiFile,
			},
			Content: []byte("#!/usr/bin/env bash\n# locked: new\nexec true\n"),
		},
	}}
	state := &State{Files: map[string]ManagedInfo{
		target: {SourceInfo: "source:config-tree (.local/bin/_index.tmpl) (multi:rg)"},
	}}

	g, ctx := taskgroup.New(logging.ContextWithLogger(t.Context(), slog.Default()), taskgroup.DefaultLimits())
	_ = g
	actions, err := NewPlanner().Plan(ctx, desired, state)
	require.NoError(t, err, "plan")
	require.Len(t, actions, 1)
	require.Equal(t, ActionUpdate, actions[0].Type)
}

func TestPlannerNoCacheForcesUpdateOnIdenticalContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "same")
	content := []byte("unchanged\n")
	require.NoError(t, os.WriteFile(target, content, 0o644), "write target")

	srcInfo := "module:x bundle:abc (same)"
	desired := []DesiredState{{
		File: &source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    "same",
				TargetBaseDir: dir,
				FileMode:      0o644,
				Info:          srcInfo,
				FileType:      source.TypeStatic,
			},
			Content: content,
		},
	}}
	state := &State{Files: map[string]ManagedInfo{
		target: {SourceInfo: srcInfo},
	}}

	base := logging.ContextWithLogger(t.Context(), slog.Default())

	// Warm path: identical managed bundle → noop
	// (taskgroup tasks use Group's root ctx; set flags before New.)
	g, ctx := taskgroup.New(base, taskgroup.DefaultLimits())
	_ = g
	actions, err := NewPlanner().Plan(ctx, desired, state)
	require.NoError(t, err, "plan warm")
	require.Len(t, actions, 1, "warm want noop, got %#v", actions)
	require.Equal(t, ActionNoop, actions[0].Type, "warm want noop, got %#v", actions)

	// no-cache: same inputs → update
	g, ctx = taskgroup.New(cmdctx.WithNoCache(base, true), taskgroup.DefaultLimits())
	_ = g
	actions, err = NewPlanner().Plan(ctx, desired, state)
	require.NoError(t, err, "plan no-cache")
	require.Len(t, actions, 1, "no-cache want update, got %#v", actions)
	require.Equal(t, ActionUpdate, actions[0].Type, "no-cache want update, got %#v", actions)
}

func TestPlannerIgnoredEqualIsNoop(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "placed.md")
	content := []byte("hello\n")
	require.NoError(t, os.WriteFile(target, content, 0o644))
	desired := []DesiredState{{
		File: &source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    "placed.md",
				TargetBaseDir: dir,
				FileMode:      0o644,
				Info:          "module:place (placed.md)",
				FileType:      source.TypeStatic,
			},
			Content: content,
		},
	}}
	state := &State{Files: map[string]ManagedInfo{}}
	p := NewPlanner()
	p.Ignore = func(path string) bool { return path == target }

	g, ctx := taskgroup.New(logging.ContextWithLogger(t.Context(), slog.Default()), taskgroup.DefaultLimits())
	_ = g
	actions, err := p.Plan(ctx, desired, state)
	require.NoError(t, err)
	require.Len(t, actions, 1, "want noop, got %#v", actions)
	require.Equal(t, ActionNoop, actions[0].Type, "want noop, got %#v", actions)
}

func TestPlannerIgnoredMissingIsCreate(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "placed.md")
	desired := []DesiredState{{
		File: &source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    "placed.md",
				TargetBaseDir: dir,
				FileMode:      0o644,
				Info:          "module:place (placed.md)",
				FileType:      source.TypeStatic,
			},
			Content: []byte("hello\n"),
		},
	}}
	p := NewPlanner()
	p.Ignore = func(path string) bool { return path == target }

	g, ctx := taskgroup.New(logging.ContextWithLogger(t.Context(), slog.Default()), taskgroup.DefaultLimits())
	_ = g
	actions, err := p.Plan(ctx, desired, &State{Files: map[string]ManagedInfo{}})
	require.NoError(t, err)
	require.Len(t, actions, 1, "want create, got %#v", actions)
	require.Equal(t, ActionCreate, actions[0].Type, "want create, got %#v", actions)
}

func TestPlannerUnmanagedEqualStillAdopts(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tracked.md")
	content := []byte("hello\n")
	require.NoError(t, os.WriteFile(target, content, 0o644))
	desired := []DesiredState{{
		File: &source.BufferFile{
			BasicFile: source.BasicFile{
				RelPathStr:    "tracked.md",
				TargetBaseDir: dir,
				FileMode:      0o644,
				Info:          "module:x (tracked.md)",
				FileType:      source.TypeStatic,
			},
			Content: content,
		},
	}}
	g, ctx := taskgroup.New(logging.ContextWithLogger(t.Context(), slog.Default()), taskgroup.DefaultLimits())
	_ = g
	actions, err := NewPlanner().Plan(ctx, desired, &State{Files: map[string]ManagedInfo{}})
	require.NoError(t, err)
	require.Len(t, actions, 1, "want adopt update, got %#v", actions)
	require.Equal(t, ActionUpdate, actions[0].Type, "want adopt update, got %#v", actions)
}
