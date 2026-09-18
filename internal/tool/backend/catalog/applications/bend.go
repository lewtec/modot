package apps

import (
	"context"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lucasew/workspaced/internal/modfile"
	"github.com/lucasew/workspaced/internal/tool/backend"
	"github.com/lucasew/workspaced/internal/tool/backend/catalog"
	"github.com/lucasew/workspaced/internal/tool/checks"
)

const bendOrigin = "https://bend-lang.com"

var ErrInvalidBendVersion = errors.New("invalid bend version")

func init() {
	catalog.RegisterTool("bend", newBend)
}

// bendTool installs Bend 2 from bend-lang.com tarballs. The official site
// only ships curl|sh, which also phones home. bun is not copied into the
// dest dir; the launcher re-enters workspaced (tool with bun) at runtime.
type bendTool struct {
	origin   string
	fetchURL func(context.Context, string) ([]byte, error)
}

type bendRelease struct {
	Ver    string `json:"ver"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func newBend() (backend.Tool, error) {
	return &bendTool{origin: bendOrigin}, nil
}

func (t *bendTool) baseURL() string {
	if o := strings.TrimRight(strings.TrimSpace(t.origin), "/"); o != "" {
		return o
	}
	return bendOrigin
}

func (t *bendTool) ListVersions(ctx context.Context) ([]string, error) {
	rel, err := t.fetchLatest(ctx)
	if err != nil {
		return nil, err
	}
	return []string{rel.Ver}, nil
}

func (t *bendTool) Install(ctx context.Context, version string, destDir string) error {
	return installFirstArtifact(ctx, version, destDir, normalizeV, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *bendTool) EnrichLockfile(entry *modfile.RenovateDependency) {
	entry.Versioning = "semver"
}

func (t *bendTool) ListArtifacts(ctx context.Context, version string) ([]backend.Artifact, error) {
	if runtime.GOOS == "windows" {
		return nil, ErrNoPlatformArtifact
	}

	v := normalizeV(version)
	rel, relErr := t.fetchLatest(ctx)
	if v == "" || v == "latest" {
		if relErr != nil {
			return nil, relErr
		}
		v = rel.Ver
	}
	if !validBendVersion(v) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidBendVersion, version)
	}

	u := fmt.Sprintf("%s/dl/%s.tar.gz", t.baseURL(), v)
	hash := ""
	if relErr == nil && rel.Ver == v {
		if rel.URL != "" {
			u = rel.URL
		}
		if rel.SHA256 != "" {
			hash = "sha256:" + rel.SHA256
		}
	}

	return []backend.Artifact{{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		URL:  u,
		Hash: hash,
	}}, nil
}

func (t *bendTool) InstallArtifact(ctx context.Context, artifact backend.Artifact, destDir string) error {
	if err := defaultInstallArtifact(ctx, artifact, destDir); err != nil {
		return err
	}
	return writeBendLauncher(destDir, "")
}

func (t *bendTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return ensureToolBinary(ctx, version, cmdName, destDir, "Bend", t.Install)
}

func (t *bendTool) InstallChecks() []checks.Check {
	return checks.Checks(checks.Binary("bend"))
}

func (t *bendTool) fetchLatest(ctx context.Context) (bendRelease, error) {
	body, err := t.fetch(ctx, t.baseURL()+"/dl/latest.json")
	if err != nil {
		return bendRelease{}, err
	}
	var rel bendRelease
	if err := jsonv2.Unmarshal(body, &rel); err != nil {
		return bendRelease{}, fmt.Errorf("decode bend latest.json: %w", err)
	}
	if !validBendVersion(rel.Ver) {
		return bendRelease{}, fmt.Errorf("%w in latest.json: %q", ErrInvalidBendVersion, rel.Ver)
	}
	return rel, nil
}

func (t *bendTool) fetch(ctx context.Context, u string) ([]byte, error) {
	if t.fetchURL != nil {
		return t.fetchURL(ctx, u)
	}
	return getBytes(ctx, u)
}

func validBendVersion(v string) bool {
	if v == "" || v[0] < '0' || v[0] > '9' || v[len(v)-1] < '0' || v[len(v)-1] > '9' {
		return false
	}
	prevDot := false
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			prevDot = false
		case r == '.':
			if prevDot {
				return false
			}
			prevDot = true
		default:
			return false
		}
	}
	return true
}

func writeBendLauncher(destDir, workspacedBin string) error {
	if strings.TrimSpace(workspacedBin) == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		workspacedBin = exe
	}
	binDir := filepath.Join(destDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
bindir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
root=$(CDPATH= cd -- "$bindir/.." && pwd)
main=""
for cand in "$root/bend2/main.ts" "$root/main.ts"; do
	if [ -f "$cand" ]; then
		main=$cand
		break
	fi
done
if [ -z "$main" ]; then
	echo "bend: missing bend2/main.ts under $root" >&2
	exit 1
fi
export BEND_NO_TELEMETRY=1
exec -a bun %s tool with bun -- bun "$main" "$@"
`, shSingleQuote(workspacedBin))
	return os.WriteFile(filepath.Join(binDir, "bend"), []byte(script), 0o755)
}

func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
