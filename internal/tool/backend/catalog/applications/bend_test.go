package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/lucasew/workspaced/internal/modfile"
)

var errUnexpectedTestURL = errors.New("unexpected test url")

func TestValidBendVersion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"2.0.5", true},
		{"2.0.0", true},
		{"v2.0.5", false},
		{"", false},
		{".", false},
		{"..", false},
		{"../evil", false},
		{"2.0.5/../x", false},
		{"2.0.5.tar.gz", false},
		{"2_0-5", false},
		{"2..0.5", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := validBendVersion(tc.in); got != tc.want {
				t.Fatalf("validBendVersion(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestBendListVersionsReadsLatestJSON(t *testing.T) {
	t.Parallel()
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)

	got, err := tool.ListVersions(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2.0.5"}
	if !slices.Equal(got, want) {
		t.Fatalf("ListVersions() = %v, want %v", got, want)
	}
}

func TestBendListArtifactsUsesLatestHash(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("bend is not packaged for windows")
	}
	const sha = "4db70e77ce1b1027f1d0e15dee025921fa794a9b415add4350ec7c64acf2775b"
	tool := newTestBend(t, fmt.Sprintf(`{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":%q}`, sha))

	arts, err := tool.ListArtifacts(t.Context(), "latest")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(arts))
	}
	if arts[0].URL != "https://bend-lang.com/dl/2.0.5.tar.gz" {
		t.Fatalf("artifact URL = %q", arts[0].URL)
	}
	if arts[0].Hash != "sha256:"+sha {
		t.Fatalf("artifact hash = %q", arts[0].Hash)
	}
}

func TestBendListArtifactsPinnedVersionOmitsUnknownHash(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("bend is not packaged for windows")
	}
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)

	arts, err := tool.ListArtifacts(t.Context(), "2.0.4")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(arts))
	}
	if arts[0].URL != "https://example.test/dl/2.0.4.tar.gz" {
		t.Fatalf("artifact URL = %q", arts[0].URL)
	}
	if arts[0].Hash != "" {
		t.Fatalf("artifact hash = %q, want empty for a pin without a published digest", arts[0].Hash)
	}
}

func TestBendListArtifactsRejectsUnsafeVersion(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("bend is not packaged for windows")
	}
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)
	_, err := tool.ListArtifacts(t.Context(), "../evil")
	if !errors.Is(err, ErrInvalidBendVersion) {
		t.Fatalf("ListArtifacts() error = %v, want ErrInvalidBendVersion", err)
	}
}

func TestBendListArtifactsWindowsUnsupported(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("windows-only assertion")
	}
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)
	_, err := tool.ListArtifacts(t.Context(), "2.0.5")
	if !errors.Is(err, ErrNoPlatformArtifact) {
		t.Fatalf("ListArtifacts() error = %v, want ErrNoPlatformArtifact", err)
	}
}

func TestWriteBendLauncher(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	if err := writeBendLauncher(dest); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dest, "bin", "bend")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("launcher mode = %v, want executable", info.Mode())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	for _, want := range []string{
		"#!/bin/sh",
		"BEND_NO_TELEMETRY=1",
		`ws=$(command -v workspaced)`,
		`exec -a bun "$ws" tool with bun -- bun "$main" "$@"`,
		"bend2/main.ts",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("launcher missing %q\n%s", want, s)
		}
	}
	if strings.Contains(s, "XDG_DATA_HOME") || strings.Contains(s, ".local/share") || strings.Contains(s, "/tmp/go-build") || strings.Contains(s, "curl") || strings.Contains(s, "bend-lang.com/ping") {
		t.Fatalf("launcher bakes a path or talks to the official installer: %s", s)
	}
}

func TestBendEnrichLockfile(t *testing.T) {
	t.Parallel()
	var entry modfile.RenovateDependency
	(&bendTool{}).EnrichLockfile(&entry)
	if entry.Versioning != "semver" {
		t.Fatalf("Versioning = %q, want semver", entry.Versioning)
	}
}

func newTestBend(t *testing.T, latestJSON string) *bendTool {
	t.Helper()
	const origin = "https://example.test"
	return &bendTool{
		origin: origin,
		fetchURL: func(_ context.Context, u string) ([]byte, error) {
			if u != origin+"/dl/latest.json" {
				return nil, fmt.Errorf("unexpected url %q: %w", u, errUnexpectedTestURL)
			}
			return []byte(latestJSON), nil
		},
	}
}
