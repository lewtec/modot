// Package tool installs external programs into ~/.local/share/workspaced/tools
// and resolves lazy tools from the workspace lockfile.
//
// Fetching, catalogs, and version directories are github.com/lewtec/lewkit/x/tool.
// This package chooses the store directory, shims, and Renovate lock rows.
package tool

import (
	"context"
	"fmt"
	"os/exec"

	kittool "github.com/lewtec/lewkit/x/tool"

	"github.com/lucasew/workspaced/internal/cmdctx"
	execdriver "github.com/lucasew/workspaced/pkg/driver/exec"
)

var (
	// ErrBinaryNotFound is returned when the binary is not in the install directory.
	ErrBinaryNotFound = kittool.ErrBinaryNotFound
	// ErrNoVersionsFound is returned when a backend lists no versions.
	ErrNoVersionsFound = kittool.ErrNoVersionsFound
	// ErrToolDirNotFound is returned when a version directory is missing.
	ErrToolDirNotFound = kittool.ErrToolDirectoryNotFound
)

// InstalledTool is one version directory in the local store.
type InstalledTool = kittool.Installed

// Manager binds the workspaced tool directory to a lewkit store.
type Manager struct {
	store *kittool.Store
}

// NewManager opens the store under GetToolsDir.
func NewManager() (*Manager, error) {
	toolsDir, err := GetToolsDir()
	if err != nil {
		return nil, err
	}
	store, err := kittool.Open(toolsDir)
	if err != nil {
		return nil, err
	}
	return &Manager{store: store}, nil
}

// Install fetches toolSpecStr into the store.
func (m *Manager) Install(ctx context.Context, toolSpecStr string) error {
	return m.store.Install(storeContext(ctx), toolSpecStr)
}

// EnsureInstalled installs toolSpecStr when needed and returns the path of cmdName.
func (m *Manager) EnsureInstalled(ctx context.Context, toolSpecStr, cmdName string) (string, error) {
	return m.store.Ensure(storeContext(ctx), toolSpecStr, cmdName)
}

// ListInstalled returns version directories present on disk.
func (m *Manager) ListInstalled() ([]InstalledTool, error) {
	return m.store.ListInstalled()
}

// ResolveLatestVersion returns the newest version the backend lists for spec.
func (m *Manager) ResolveLatestVersion(ctx context.Context, spec kittool.Spec) (string, error) {
	backend, err := kittool.Get(spec.Backend)
	if err != nil {
		return "", err
	}
	installed, err := backend.Tool(spec.Package)
	if err != nil {
		return "", err
	}
	versions, err := installed.ListVersions(ctx)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", ErrNoVersionsFound
	}
	return versions[0], nil
}

// EnsureAndRun installs cmdName from toolSpecStr and returns a command ready to start.
func EnsureAndRun(ctx context.Context, toolSpecStr, cmdName string, args ...string) (*exec.Cmd, error) {
	manager, err := NewManager()
	if err != nil {
		return nil, fmt.Errorf("create tool manager: %w", err)
	}
	binPath, err := manager.EnsureInstalled(ctx, toolSpecStr, cmdName)
	if err != nil {
		return nil, fmt.Errorf("ensure tool installed: %w", err)
	}
	return execdriver.Run(ctx, binPath, args...)
}

func storeContext(ctx context.Context) context.Context {
	if cmdctx.IsNoCache(ctx) {
		ctx = kittool.WithNoCache(ctx)
	}
	if cmdctx.IsDryRun(ctx) {
		ctx = kittool.WithDryRun(ctx)
	}
	return ctx
}
