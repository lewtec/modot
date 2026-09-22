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
	kittool "github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	apps "github.com/lewtec/lewkit/x/tool/registry/applications"
	"github.com/lucasew/workspaced/pkg/logging"
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
				t.Errorf("taskgroup: %v", err)
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
			if err != nil {
				t.Fatalf("NewTool(%q): %v", name, err)
			}
			checker, ok := installed.(kittool.Checker)
			if !ok {
				return
			}
			if len(checker.InstallChecks()) == 0 {
				t.Fatalf("%q implements Checker but InstallChecks() is empty", name)
			}
		})
	}
}

func TestRegistryInstall(t *testing.T) {
	if os.Getenv("WORKSPACED_TEST_TOOL_INSTALL") != "1" {
		t.Skip("set WORKSPACED_TEST_TOOL_INSTALL=1 (or mise run test:registry-install) to run registry install checks")
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
				t.Fatalf("NewTool(%q): %v", name, err)
			}
			checker, ok := installed.(kittool.Checker)
			if !ok || len(checker.InstallChecks()) == 0 {
				t.Skip("no install checks")
			}

			ctx, wait := testInstallContext(t)
			defer wait()

			versions, err := installed.ListVersions(ctx)
			if err != nil {
				reportInstallFailure(name, fmt.Sprintf("ListVersions: %v", err))
				t.Fatalf("ListVersions: %v", err)
			}
			if len(versions) == 0 {
				reportInstallFailure(name, "ListVersions returned no versions")
				t.Fatal("ListVersions returned no versions")
			}

			dest := t.TempDir()
			if err := installed.Install(ctx, versions[0], dest); err != nil {
				if errors.Is(err, github.ErrNoArtifact) || errors.Is(err, apps.ErrNoPlatformArtifact) {
					t.Skipf("no artifact for %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
				}
				reportInstallFailure(name, fmt.Sprintf("Install(%q): %v", versions[0], err))
				t.Fatalf("Install(%q): %v", versions[0], err)
			}
			if fixer, ok := installed.(kittool.Fixer); ok {
				if err := fixer.Fix(ctx, dest); err != nil {
					reportInstallFailure(name, fmt.Sprintf("Fix: %v", err))
					t.Fatalf("Fix: %v", err)
				}
			}
			if err := kittool.RunChecks(ctx, dest, installed); err != nil {
				reportInstallFailure(name, fmt.Sprintf("RunChecks: %v", err))
				t.Fatalf("RunChecks: %v", err)
			}
		})
	}
}
