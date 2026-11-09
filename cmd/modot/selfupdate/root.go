package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/taskgroup"
	lewtool "github.com/lewtec/lewkit/x/tool"
	githubprov "github.com/lewtec/lewkit/x/tool/github"
	envdriver "github.com/lewtec/modot/internal/driver/env"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
	"github.com/lewtec/modot/internal/miseutil"
	"github.com/lewtec/modot/internal/selfbin"
	"github.com/lewtec/modot/internal/version"

	"github.com/lewtec/lewkit/x/cmd"
)

var (
	ErrGoVersionUnknown     = errors.New("could not determine Go version from build info")
	ErrNoVersionsFound      = errors.New("no versions found")
	ErrArtifactToolRequired = errors.New("github tool does not support ArtifactTool (needed for selfupdate)")
	ErrNoArtifactFound      = errors.New("no artifact found for current platform")
	ErrNoBinaryFound        = errors.New("no binary found")
)

type Command struct {
	Force cmd.Flag `long:"force" help:"Force update even if version matches (GitHub only)"`
}

func (Command) Description() string {
	return "Update modot binary"
}

func (c *Command) Run(ctx context.Context) error {
	msg := "downloading from GitHub"
	srcPath, err := findSourcePath(ctx)
	if err != nil {
		return err
	}
	if srcPath != "" {
		msg = "compiling from source"
	}

	// Control: github/httpclient and the source build nest limited-pool work.
	// Do not Unit here — GitHub installs already own a fetch bar.
	taskgroup.Go(ctx, "self-update", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update(msg)
		return runSelfUpdate(ctx, c.Force.Value(), s)
	})

	return nil
}

// ============================================================================
// Main update flow
// ============================================================================

func runSelfUpdate(ctx context.Context, force bool, s *taskgroup.Status) error {
	// Try source build first (dev mode - always rebuilds)
	srcPath, err := findSourcePath(ctx)
	if err != nil {
		return err
	}
	if srcPath != "" {
		slog.Info("building from source (always rebuilds)", "path", srcPath)
		return buildAndInstallFromSource(ctx, srcPath, s)
	}

	// Fallback to GitHub provider (checks version unless --force)
	return updateFromGitHub(ctx, force, s)
}

// ============================================================================
// Source build strategy
// ============================================================================

func buildAndInstallFromSource(ctx context.Context, srcPath string, s *taskgroup.Status) error {
	// Install to fixed location (not versioned); real home, not Termux /home chroot view.
	installDir, installPath, err := selfbin.InstallPaths(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return err
	}
	tmpOut, err := os.CreateTemp(installDir, "modot.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpOut.Name()
	if err := tmpOut.Close(); err != nil {
		return err
	}
	defer func() {
		if err := func() error {
			if err := os.Remove(tmpPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}(); err != nil {
			slog.Error("unexpected error", "op", "remove",

				"path", tmpPath, "error", err)
		}
	}()

	toolchain, err := resolveGoToolchain(ctx)
	if err != nil {
		return fmt.Errorf("resolve Go toolchain: %w", err)
	}
	slog.Info("building from source", "path", srcPath, "go", toolchain.version, "via", toolchain.name)
	slog.Debug("go toolchain", "argv", toolchain.buildArgs(tmpPath))

	prog := newBuildProgress(s)
	if s != nil {
		s.Progress(0, 1)
		s.Update("compiling")
	}
	if n, err := countBuildPackages(ctx, toolchain, srcPath); err == nil && n > 0 {
		prog.total = n
		if s != nil {
			s.Progress(0, n)
		}
	}

	buildCmd, err := execdriver.Run(ctx, toolchain.path, toolchain.buildArgs(tmpPath)...)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	buildCmd.Env = toolchain.env(buildCmd.Env)
	buildCmd.Dir = srcPath
	if s == nil {
		// No bar to draw the packages on: stream them instead of going silent.
		prog.echo = os.Stderr
	}
	buildCmd.Stdout = prog
	buildCmd.Stderr = prog

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w\n%s", err, prog.tail())
	}
	prog.finish("installing")
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("set permissions on built binary: %w", err)
	}
	if err := os.Rename(tmpPath, installPath); err != nil {
		return fmt.Errorf("install built binary: %w", err)
	}

	slog.Info("build completed", "path", installPath)
	return selfbin.EnsureModotShim(ctx, installPath)
}

// ============================================================================
// GitHub provider strategy
// ============================================================================

func updateFromGitHub(ctx context.Context, force bool, s *taskgroup.Status) error {
	// Use the exposed constructor directly. This works even without the old
	// detailed methods on the thin Provider interface, and demonstrates how
	// a future registry backend (or other code) can obtain a github Tool.
	t, err := githubprov.NewTool("lewtec/modot")
	if err != nil {
		return err
	}

	// Get latest version via the Tool
	versions, err := t.ListVersions(ctx)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return ErrNoVersionsFound
	}

	latestVersion := versions[0]
	normalizedLatest := strings.TrimPrefix(latestVersion, "v")

	// Check if update is needed (unless --force)
	if !force {
		currentVersion := version.Version()

		if currentVersion == normalizedLatest {
			slog.Info("already at latest version", "version", currentVersion)
			return nil
		}

		slog.Info("updating", "current", currentVersion, "latest", normalizedLatest)
	} else {
		slog.Info("forcing update", "version", latestVersion)
	}

	// Use ArtifactTool + the shared SelectArtifact for platform selection.
	at, ok := t.(lewtool.ArtifactTool)
	if !ok {
		return ErrArtifactToolRequired
	}

	artifacts, err := at.ListArtifacts(ctx, latestVersion)
	if err != nil {
		return err
	}

	// Standard platform selection (same logic used by tool installs etc.).
	// The "modot" hint helps disambiguate when a release has multiple
	// assets for the same OS/arch.
	artifact := lewtool.SelectArtifact(artifacts, runtime.GOOS, runtime.GOARCH, "modot")
	if artifact == nil {
		available := []string{}
		for _, a := range artifacts {
			available = append(available, fmt.Sprintf("%s/%s", a.OS, a.Arch))
		}
		return fmt.Errorf("%w: %s/%s (available: %v)", ErrNoArtifactFound, runtime.GOOS, runtime.GOARCH, available)
	}

	// Install to fixed location (not versioned); real home, not Termux /home chroot view.
	installDir, _, err := selfbin.InstallPaths(ctx)
	if err != nil {
		return err
	}
	tmpDir := filepath.Join(installDir, ".tmp-"+normalizedLatest)

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	defer func() {
		if err := func() error { return os.RemoveAll(tmpDir) }(); err != nil {
			slog.Error("unexpected error", "op", "remove_all", "path", tmpDir, "error", err)
		}
	}()

	slog.Info("downloading from GitHub", "version", latestVersion, "os", artifact.OS, "arch", artifact.Arch)
	if s != nil {
		s.Update("downloading")
	}
	if err := at.InstallArtifact(ctx, *artifact, tmpDir); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	modotBin, err := findBinary(tmpDir)
	if err != nil {
		return fmt.Errorf("modot binary not found in downloaded archive: %w", err)
	}

	targetName := "modot"
	if runtime.GOOS == "windows" {
		targetName = "modot.exe"
	}
	installPath := filepath.Join(installDir, targetName)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return err
	}

	if err := os.Rename(modotBin, installPath); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}

	if err := os.Chmod(installPath, 0755); err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}

	slog.Info("download completed", "path", installPath)
	return selfbin.EnsureModotShim(ctx, installPath)
}

func findBinary(dir string) (string, error) {
	// 1. Check standard names (strict first)
	targets := []string{"modot", "modot.exe"}
	for _, t := range targets {
		path := filepath.Join(dir, t)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		// Check bin/ subdirectory
		path = filepath.Join(dir, "bin", t)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 2. Scan for any executable-looking file
	// This fallback handles cases where:
	// - Binary has a suffix (e.g. modot-linux-amd64) and extraction didn't rename it
	// - Binary name is different from expected
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Ignore common non-binary files
		if strings.HasPrefix(name, ".") ||
			strings.HasSuffix(name, ".sha256") ||
			strings.HasSuffix(name, ".md") ||
			strings.HasSuffix(name, ".txt") ||
			name == "LICENSE" {
			continue
		}

		// On Unix, check executable bit
		if runtime.GOOS != "windows" {
			info, err := e.Info()
			if err == nil && info.Mode()&0111 != 0 {
				return filepath.Join(dir, name), nil
			}
		} else {
			// On Windows, check extension
			if strings.HasSuffix(strings.ToLower(name), ".exe") {
				return filepath.Join(dir, name), nil
			}
		}
	}

	// 3. Last resort: if there is only one file and it's not excluded above, pick it
	// This covers Linux binaries that might not have +x set yet (though they should)
	var candidates []string
	for _, e := range entries {
		if !e.IsDir() {
			name := e.Name()
			if !strings.HasPrefix(name, ".") &&
				!strings.HasSuffix(name, ".sha256") &&
				!strings.HasSuffix(name, ".md") &&
				!strings.HasSuffix(name, ".txt") &&
				name != "LICENSE" {
				candidates = append(candidates, filepath.Join(dir, name))
			}
		}
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	return "", fmt.Errorf("%w: %s", ErrNoBinaryFound, dir)
}

// buildProgress counts `go build -v` package lines onto a Status bar.
type buildProgress struct {
	s *taskgroup.Status
	// echo receives the lines when there is no status bar.
	echo  io.Writer
	mu    sync.Mutex
	buf   []byte
	n     int64
	total int64
	last  []string
}

func newBuildProgress(s *taskgroup.Status) *buildProgress {
	return &buildProgress{s: s}
}

func (w *buildProgress) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSpace(string(bytes.TrimSuffix(w.buf[:i], []byte{'\r'})))
		w.buf = w.buf[i+1:]
		if line == "" {
			continue
		}
		w.last = append(w.last, line)
		if len(w.last) > 8 {
			w.last = w.last[len(w.last)-8:]
		}
		w.n++
		total := w.total
		if total < w.n {
			total = w.n
			w.total = total
		}
		if w.s != nil && total > 0 {
			w.s.Progress(w.n, total)
			w.s.Update(line)
		} else if w.echo != nil {
			if _, err := fmt.Fprintln(w.echo, line); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}

func (w *buildProgress) tail() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.Join(w.last, "\n")
}

func (w *buildProgress) finish(msg string) {
	w.mu.Lock()
	total := w.total
	s := w.s
	w.mu.Unlock()
	if s == nil {
		return
	}
	if total < 1 {
		total = 1
	}
	s.Progress(total, total)
	s.Update(msg)
}

// goToolchain is how to invoke the Go build for a source build.
//
// A PATH `go` runs directly. Otherwise mise provides the toolchain:
// `mise exec go@<version> -- go <args>`. The prefix is data, never spliced
// from a user string, so a Go flag can never be parsed as a mise flag.
type goToolchain struct {
	// name and version are for logging.
	name    string
	version string
	path    string
	// prefix is prepended to the `go` arguments.
	prefix []string
	// pin pins GOTOOLCHAIN in the child: local when PATH's go is at least the
	// go.mod directive, auto when the directive must be fetched.
	pin string
}

// args copies prefix before a subcommand.
func (t goToolchain) args() []string {
	args := make([]string, len(t.prefix), len(t.prefix)+7)
	copy(args, t.prefix)
	// After mise's `--` an environment entry would be run as a program, so the
	// prefix that names mise must name `go` again.
	if len(t.prefix) > 0 {
		args = append(args, "go")
	}
	return args
}

// env returns the toolchain environment with GOTOOLCHAIN pinned, when it is.
//
// The exec driver leaves Cmd.Env nil, which os/exec reads as "inherit"; go
// needs the inherited environment (HOME, GOPATH, PATH) whatever the pin.
func (t goToolchain) env(base []string) []string {
	if base == nil {
		base = os.Environ()
	}
	if t.pin == "" {
		return base
	}
	return append(slices.Clone(base), "GOTOOLCHAIN="+t.pin)
}

func (t goToolchain) buildArgs(outPath string) []string {
	return append(t.args(), "build", "-v", "-o", outPath, "./cmd/modot")
}

func (t goToolchain) listArgs() []string {
	return append(t.args(), "list", "-e", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./cmd/modot")
}

func (t goToolchain) versionArgs() []string {
	return append(t.args(), "env", "GOVERSION")
}

// GOTOOLCHAIN modes a source build pins. Empty means the go's own default.
const (
	pinLocal = "local"
	pinAuto  = "auto"
)

// goInPath returns PATH's `go` and its GOVERSION.
func goInPath(ctx context.Context) (goToolchain, bool) {
	path, err := execdriver.Which(ctx, "go")
	if err != nil {
		return goToolchain{}, false
	}
	tc := goToolchain{name: "go", path: path}
	if got, err := goVersionArg(ctx, tc); err == nil && got != "" {
		tc.version = got
		return tc, true
	}
	return goToolchain{}, false
}

// resolveGoToolchain picks a Go toolchain for a source build.
//
// `mise exec` parses every argument before the `--`, so `go build -o <path>`
// itself must never run as a mise option: a prefix missing its `--` turns the
// `-o` flag into a mise flag ("a value is required for '--env <ENV>'").
//
// PATH's `go` wins when it is at least this module's Go directive; a slightly
// older one still wins because GOTOOLCHAIN can fetch the directive. mise is
// only a fallback: the project's mise.toml lists micro tools (gopls, goreleaser,
// svu), not the Go toolchain.
func resolveGoToolchain(ctx context.Context) (goToolchain, error) {
	need := getGoVersion()
	if need == "" {
		return goToolchain{}, ErrGoVersionUnknown
	}
	if tc, ok := goInPath(ctx); ok {
		if goVersionAtLeast(tc.version, need) {
			tc.pin = pinLocal
			return tc, nil
		}
		if goSupportsToolchainSwitch(tc) {
			tc.pin = pinAuto
			return tc, nil
		}
	}

	misePath, err := miseutil.Ensure(ctx)
	if err != nil {
		return goToolchain{}, fmt.Errorf("ensure mise: %w", err)
	}
	tc := goToolchain{
		name:    "mise",
		version: need,
		path:    misePath,
		prefix:  []string{"exec", "go@" + need, "--"},
		pin:     pinLocal,
	}
	got, err := goVersionArg(ctx, tc)
	if err != nil {
		return goToolchain{}, fmt.Errorf("mise Go %s: %w", need, err)
	}
	tc.version = got
	return tc, nil
}

// goVersionArg runs `go env GOVERSION` for a toolchain.
//
// A mise prefix may install the tool first, so the run keeps stdin away from
// the tool output and reports what the child printed when it fails.
func goVersionArg(ctx context.Context, tc goToolchain) (string, error) {
	cmd, err := execdriver.Run(ctx, tc.path, tc.versionArgs()...)
	if err != nil {
		return "", err
	}
	cmd.Env = tc.env(cmd.Env)
	cmd.Stdin = nil
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s %s: %w: %s", filepath.Base(tc.path), strings.Join(tc.versionArgs(), " "), err, detail)
	}
	// Only the value counts: a GOTOOLCHAIN switch may still print a notice.
	return lastLine(stdout.String()), nil
}

// lastLine returns the last non-empty line, trimmed.
func lastLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

// goSupportsToolchainSwitch reports whether a go can fetch the go.mod directive
// through GOTOOLCHAIN (go 1.21 and newer).
func goSupportsToolchainSwitch(tc goToolchain) bool {
	major, minor := parseGoVersion(strings.TrimPrefix(tc.version, "go"))
	return major > 1 || (major == 1 && minor >= 21)
}

// goVersionAtLeast reports whether a `go env GOVERSION` value (such as go1.27.0)
// is at least want (such as 1.27.0).
func goVersionAtLeast(version, want string) bool {
	version = strings.TrimPrefix(version, "go")
	// Drop devel and release-candidate suffixes: GOVERSION "go1.27devel" and
	// "go1.28rc1" both work for a module that asks for 1.27.
	version = strings.TrimSuffix(version, "devel")
	version = strings.TrimSuffix(version, "rc")

	gotMajor, gotMinor := parseGoVersion(version)
	wantMajor, wantMinor := parseGoVersion(want)
	if gotMajor != wantMajor {
		return gotMajor > wantMajor
	}
	return gotMinor >= wantMinor
}

// parseGoVersion reads the major and minor of a version (1.27.0, 1.27devel).
func parseGoVersion(s string) (int, int) {
	parts := strings.SplitN(s, ".", 3)
	var major, minor int
	if len(parts) > 0 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}

func countBuildPackages(ctx context.Context, toolchain goToolchain, srcPath string) (int64, error) {
	cmd, err := execdriver.Run(ctx, toolchain.path, toolchain.listArgs()...)
	if err != nil {
		return 0, err
	}
	cmd.Env = toolchain.env(cmd.Env)
	var stdout, stderr bytes.Buffer
	cmd.Dir = srcPath
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("go list: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var n int64
	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n, nil
}

func findSourcePath(ctx context.Context) (string, error) {
	var candidates []string

	// 1. ~/.config/modot/src/
	configDir, err := envdriver.GetConfigDir(ctx)
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	candidates = append(candidates, filepath.Join(configDir, "src"))

	// 2. $DOTFILES/modot/
	dotfilesRoot, err := envdriver.GetDotfilesRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("dotfiles root: %w", err)
	}
	candidates = append(candidates, filepath.Join(dotfilesRoot, "modot"))

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// No candidate exists on disk; caller falls back to GitHub update.
	return "", nil
}

func getGoVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	version := info.GoVersion
	if len(version) > 2 && version[0] == 'g' && version[1] == 'o' {
		return version[2:]
	}
	return version
}
