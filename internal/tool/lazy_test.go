package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/lucasew/workspaced/internal/configcue"
	"github.com/lucasew/workspaced/internal/modfile"
	_ "github.com/lucasew/workspaced/pkg/driver/env/native"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshLazyToolLocksPreservesExistingLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	workspaceRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workspaceRoot, "workspaced.cue"), `package workspaced

lazy_tools: {
	gh: {
		ref: "github:cli/cli"
		bins: ["gh"]
	}
}
`)

	// Pin every lazy tool that the codebase prelude injects (plus gh from the
	// test cue) so refresh has nothing to resolve and must leave the lock untouched.
	writeTestFile(t, filepath.Join(workspaceRoot, "workspaced.lock.json"), `{
  "dependencies": [
    {"kind": "tool", "ref": "github:cli/cli", "currentValue": "0.1.0", "depName": "cli/cli", "datasource": "github-releases"},
    {"kind": "tool", "ref": "github:golangci/golangci-lint", "currentValue": "1.0.0", "depName": "golangci/golangci-lint", "datasource": "github-releases"},
    {"kind": "tool", "ref": "registry:shellcheck", "currentValue": "0.1.0", "depName": "shellcheck", "datasource": "github-releases"},
    {"kind": "tool", "ref": "github:astral-sh/ruff", "currentValue": "0.1.0", "depName": "astral-sh/ruff", "datasource": "github-releases"},
    {"kind": "tool", "ref": "github:rhysd/actionlint", "currentValue": "0.1.0", "depName": "rhysd/actionlint", "datasource": "github-releases"},
    {"kind": "tool", "ref": "registry:biome", "currentValue": "0.1.0", "depName": "biome", "datasource": "github-releases"},
    {"kind": "tool", "ref": "registry:nodejs", "currentValue": "0.1.0", "depName": "nodejs", "datasource": "github-releases"},
    {"kind": "tool", "ref": "registry:mise", "currentValue": "0.1.0", "depName": "jdx/mise", "datasource": "github-releases"},
    {"kind": "tool", "ref": "registry:resvg", "currentValue": "0.1.0", "depName": "linebender/resvg", "datasource": "github-releases"},
    {"kind": "tool", "ref": "github:lucasew/ci-status", "currentValue": "0.1.0", "depName": "lucasew/ci-status", "datasource": "github-releases"},
    {"kind": "tool", "ref": "mise:go:golang.org/x/vuln/cmd/govulncheck", "currentValue": "0.1.0"}
  ]
}
`)

	spec, err := lewtool.Parse("github:cli/cli")
	require.NoError(t, err)
	binPath := filepath.Join(home, ".local", "share", "workspaced", "tools", spec.Directory(), "2.89.0", "bin", "gh")
	writeTestFile(t, binPath, "#!/bin/sh\nexit 0\n")
	require.NoError(t, os.Chmod(binPath, 0o755))

	g, ctx := taskgroup.New(logging.NewWriterContext(t.Output()), taskgroup.DefaultLimits())
	t.Cleanup(func() {
		if err := g.Wait(); err != nil && !t.Failed() {
			assert.NoError(t, err, "group wait")
		}
	})
	cfg, err := configcue.LoadForWorkspace(ctx, workspaceRoot)
	require.NoError(t, err)

	ws := modfile.NewWorkspace(workspaceRoot)
	before, err := os.ReadFile(ws.SumPath())
	require.NoError(t, err)

	updated, err := RefreshLazyToolLocks(ctx, ws, cfg)
	require.NoError(t, err)
	require.Equal(t, 0, updated)

	after, err := os.ReadFile(ws.SumPath())
	require.NoError(t, err)
	require.Equal(t, before, after, "lockfile changed unexpectedly")
}

func TestApplyLiveToolEnrichmentIdempotent(t *testing.T) {
	t.Parallel()

	sum := &modfile.SumFile{
		Dependencies: []modfile.RenovateDependency{{
			Kind:         "tool",
			Ref:          "github:cli/cli",
			DepName:      "cli/cli",
			CurrentValue: "v2.95.0",
			Datasource:   "github-releases",
		}},
	}
	live := staticEnrichTool{
		depName:    "cli/cli",
		datasource: "github-releases",
	}
	require.False(t, applyLiveToolEnrichment(sum, "github:cli/cli", "v2.95.0", live), "expected no change when enrichment matches existing row")
	require.True(t, applyLiveToolEnrichment(sum, "github:cli/cli", "v2.95.0", staticEnrichTool{
		depName:     "cli/cli",
		datasource:  "github-releases",
		versioning:  "semver",
		extractVers: `^v(?<version>\d+)`,
	}), "expected change when enrichment adds metadata")
	require.Equal(t, "semver", sum.Dependencies[0].Versioning)
	require.True(t, applyLiveToolEnrichment(sum, "github:other/other", "1.0.0", nil), "expected change when creating missing row")
	require.False(t, applyLiveToolEnrichment(sum, "github:other/other", "1.0.0", nil), "expected create to be idempotent on second call")
}

type staticEnrichTool struct {
	depName     string
	datasource  string
	versioning  string
	extractVers string
}

func (t staticEnrichTool) ListVersions(context.Context) ([]string, error) { return nil, nil }
func (t staticEnrichTool) Install(context.Context, string, string) error  { return nil }
func (t staticEnrichTool) Pin() lewtool.Pin {
	return lewtool.Pin{
		Name:           t.depName,
		Datasource:     t.datasource,
		Versioning:     t.versioning,
		ExtractVersion: t.extractVers,
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
