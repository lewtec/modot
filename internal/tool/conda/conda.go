// Package conda is the conda channel backend.
//
// A spec conda:ripgrep@15.2.0 installs that package from conda-forge.
// conda:bioconda/samtools and conda:defaults::curl select another channel.
// conda:https://example.test/channel::pkg uses that repository directly.
//
// Versions and dependency metadata come from the channel: the anaconda.org
// package index when the channel publishes one, otherwise repodata.json
// (zst, then json) on the repository. Artifacts are downloaded from the
// channel repository itself.
package conda

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	lewtool "github.com/lewtec/lewkit/x/tool"
)

// Backend installs packages from conda repositories.
type Backend struct {
	fetch    fetcher
	download downloader
	subdir   string
}

type downloader func(ctx context.Context, rawURL, dest, hash string, size int64) error

// Name returns the backend id used in specs.
func (backend *Backend) Name() string { return "conda" }

// Tool returns the conda package named by ref.
func (backend *Backend) Tool(ref string) (lewtool.Tool, error) {
	parsed, err := parseRef(ref)
	if err != nil {
		return nil, err
	}
	subdir := backend.subdir
	if subdir == "" {
		subdir, err = currentSubdir()
		if err != nil {
			return nil, err
		}
	}
	fetch := backend.fetch
	if fetch == nil {
		fetch = defaultFetch
	}
	save := backend.download
	if save == nil {
		save = defaultDownload
	}
	return &condaTool{ref: parsed, subdir: subdir, fetch: fetch, download: save}, nil
}

// condaTool is one package in one channel.
type condaTool struct {
	ref      parsedRef
	subdir   string
	fetch    fetcher
	download downloader
}

// ListVersions returns channel versions for this platform, newest first.
func (installed *condaTool) ListVersions(ctx context.Context) ([]string, error) {
	recs, err := newIndex(installed.fetch).records(ctx, installed.ref.Channel, installed.ref.Name, installed.subdir)
	if err != nil {
		return nil, err
	}
	versions := uniqueVersions(recs, installed.subdir)
	if len(versions) == 0 {
		return nil, fmt.Errorf("%w: %s on %s", ErrNoBuild, installed.ref.Name, installed.subdir)
	}
	return versions, nil
}

// Install resolves the package and its run dependencies, then extracts them
// into destination. version may be a conda version, version=build, or latest.
func (installed *condaTool) Install(ctx context.Context, version, destination string) error {
	versionSpec, build := requestedSpec(version)
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	prefix := installPrefix(destination)
	tmp, err := os.MkdirTemp("", "modot-conda-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	slog.InfoContext(ctx, "installing conda packages", "ref", installed.ref.Name, "channel", installed.ref.Channel.Name, "prefix", prefix)
	ix := newIndex(installed.fetch)
	recs, err := ix.closure(ctx, installed.ref.Channel, installed.ref.Name, versionSpec, build, installed.subdir, func(ctx context.Context, rec record) error {
		return installed.download(ctx, rec.URL, filepath.Join(tmp, archiveName(rec)), hashOf(rec), rec.Size)
	})
	if err != nil {
		return err
	}
	for _, rec := range recs {
		archivePath := filepath.Join(tmp, archiveName(rec))
		if err := extractPackage(archivePath, destination, prefix); err != nil {
			return fmt.Errorf("extract %s-%s: %w", rec.Name, rec.Version, err)
		}
	}
	return nil
}

// Pin names the channel package for the lockfile.
func (installed *condaTool) Pin() lewtool.Pin {
	return lewtool.Pin{
		Name:       installed.ref.Channel.Name + "/" + installed.ref.Name,
		Datasource: "conda",
		Versioning: "conda",
	}
}

func archiveName(rec record) string {
	return rec.Subdir + "-" + rec.Filename
}

func hashOf(rec record) string {
	switch {
	case rec.SHA256 != "":
		if strings.Contains(rec.SHA256, ":") {
			return rec.SHA256
		}
		return "sha256:" + rec.SHA256
	case rec.MD5 != "":
		if strings.Contains(rec.MD5, ":") {
			return rec.MD5
		}
		return "md5:" + rec.MD5
	default:
		return ""
	}
}

func defaultDownload(ctx context.Context, rawURL, dest, hash string, size int64) error {
	opts := lewtool.DownloadOptions{Size: size, Hash: hash}
	if err := lewtool.DownloadFile(ctx, rawURL, dest, opts); err != nil {
		return fmt.Errorf("download %s: %w", rawURL, err)
	}
	return nil
}
