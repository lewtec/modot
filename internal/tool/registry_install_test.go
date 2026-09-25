package tool_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	apps "github.com/lewtec/lewkit/x/tool/registry/applications"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stepSummaryMu sync.Mutex

func appendStepSummary(s string) {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return
	}
	stepSummaryMu.Lock()
	defer stepSummaryMu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, writeErr := f.WriteString(s)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return
	}
}

func reportInstallFailure(name, msg string) {
	appendStepSummary(fmt.Sprintf("### `%s`\n\n```\n%s\n```\n\n", name, strings.TrimSpace(msg)))
}

func testInstallContext(t *testing.T) (ctx context.Context, wait func()) {
	t.Helper()
	base := logging.NewWriterContext(t.Output())
	group, ctx := taskgroup.New(base, taskgroup.DefaultLimits())
	return ctx, func() {
		done := make(chan error, 1)
		go func() { done <- group.Wait() }()
		select {
		case err := <-done:
			if err != nil && !t.Failed() {
				assert.NoError(t, err, "taskgroup")
			}
		case <-t.Context().Done():
		}
	}
}

func TestRegistryInstallChecksDeclared(t *testing.T) {
	t.Parallel()
	for _, name := range registry.ListTools() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			installed, err := registry.NewTool(name)
			require.NoError(t, err)
			checker, ok := installed.(lewtool.Checker)
			if !ok {
				return
			}
			require.NotEmpty(t, checker.InstallChecks(), "%q implements Checker but InstallChecks() is empty", name)
		})
	}
}

func TestRegistryInstall(t *testing.T) {
	if os.Getenv("MODOT_TEST_TOOL_INSTALL") != "1" {
		t.Skip("set MODOT_TEST_TOOL_INSTALL=1 (or mise run test:registry-install) to run registry install checks")
	}

	target := os.Getenv("TARGET")
	if target == "" {
		target = runtime.GOOS + "/" + runtime.GOARCH
	}
	appendStepSummary(fmt.Sprintf("## Registry install (`%s`)\n\n", target))

	for _, name := range registry.ListTools() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			installed, err := registry.NewTool(name)
			if err != nil {
				reportInstallFailure(name, fmt.Sprintf("NewTool: %v", err))
				require.NoError(t, err)
			}
			checker, ok := installed.(lewtool.Checker)
			if !ok || len(checker.InstallChecks()) == 0 {
				t.Skip("no install checks")
			}

			ctx, wait := testInstallContext(t)
			defer wait()

			versions, err := installed.ListVersions(ctx)
			if err != nil {
				reportInstallFailure(name, fmt.Sprintf("ListVersions: %v", err))
				require.NoError(t, err)
			}
			if len(versions) == 0 {
				reportInstallFailure(name, "ListVersions returned no versions")
				require.Fail(t, "ListVersions returned no versions")
			}

			dest := t.TempDir()
			if err := installed.Install(ctx, versions[0], dest); err != nil {
				if errors.Is(err, github.ErrNoArtifact) || errors.Is(err, apps.ErrNoPlatformArtifact) {
					t.Skipf("no artifact for %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
				}
				reportInstallFailure(name, fmt.Sprintf("Install(%q): %v", versions[0], err))
				require.NoError(t, err)
			}
			if fixer, ok := installed.(lewtool.Fixer); ok {
				if err := fixer.Fix(ctx, dest); err != nil {
					reportInstallFailure(name, fmt.Sprintf("Fix: %v", err))
					require.NoError(t, err)
				}
			}
			if err := lewtool.RunChecks(ctx, dest, installed); err != nil {
				reportInstallFailure(name, fmt.Sprintf("RunChecks: %v", err))
				require.NoError(t, err)
			}
		})
	}
}
