package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/squashfs"
	tarfs "github.com/lewtec/lewkit/x/fs/tar"
	zipfs "github.com/lewtec/lewkit/x/fs/zip"
	lewpath "github.com/lewtec/lewkit/x/path"

	"github.com/lucasew/workspaced/internal/archive"
	"github.com/lucasew/workspaced/internal/atomicfile"
	"github.com/lucasew/workspaced/internal/constants"
	"github.com/lucasew/workspaced/internal/tool/backend"
	"github.com/lucasew/workspaced/pkg/driver"
	"github.com/lucasew/workspaced/pkg/driver/fetchurl"
	"github.com/lucasew/workspaced/pkg/driver/httpclient"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/lucasew/workspaced/pkg/taskgroup"
	"io/fs"
)

var (
	ErrEmptyDownloadURL = errors.New("download URL cannot be empty")
	ErrNoDownloadURLs   = errors.New("no download URLs provided")
)

type DownloadOptions struct {
	Hash             string
	Size             int64
	Mode             os.FileMode
	ConfigureRequest func(*http.Request)
}

func InstallArtifact(ctx context.Context, artifact backend.Artifact, destDir string, opts DownloadOptions) error {
	if opts.Hash == "" {
		opts.Hash = artifact.Hash
	}
	if opts.Size <= 0 {
		opts.Size = artifact.Size
	}

	tmpDir := destDir + ".tmp"
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	defer logging.RunCleanup(ctx, "remove_all", func() error { return os.RemoveAll(tmpDir) })

	downloadPath := filepath.Join(tmpDir, filepath.Base(artifact.URL))
	if err := DownloadFile(ctx, artifact.URL, downloadPath, opts); err != nil {
		return err
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return err
	}
	if err := Extract(ctx, downloadPath, extractDir); err != nil {
		return fmt.Errorf("extract %s: %w", filepath.Base(artifact.URL), err)
	}
	if err := archive.StripTopLevelDir(extractDir); err != nil {
		return err
	}
	if err := MoveContents(extractDir, destDir); err != nil {
		return err
	}
	return NormalizeInstalledBinaries(destDir)
}

func DownloadFile(ctx context.Context, url, dest string, opts DownloadOptions) error {
	if strings.TrimSpace(url) == "" {
		return ErrEmptyDownloadURL
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	var fetchErr error
	if opts.Hash != "" {
		fetchErr = downloadWithFetchurl(ctx, url, dest, opts)
		if fetchErr == nil {
			return nil
		}
		logger := logging.GetLogger(ctx)
		logger.Warn("fetchurl verified download failed, falling back to direct http download", "url", url, "err", fetchErr)
		tmp := atomicfile.SiblingTemp(dest)
		if rmErr := os.Remove(tmp); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			logging.ReportError(ctx, rmErr, "path", tmp)
		}
	}

	if err := downloadDirect(ctx, url, dest, opts); err != nil {
		if fetchErr != nil {
			return fmt.Errorf("verified download failed: %w; direct download failed: %w", fetchErr, err)
		}
		return err
	}
	return nil
}

func DownloadFirst(ctx context.Context, urls []string, dest string, opts DownloadOptions) error {
	var errs []string
	for _, url := range urls {
		if strings.TrimSpace(url) == "" {
			continue
		}
		if err := DownloadFile(ctx, url, dest, opts); err == nil {
			return nil
		} else {
			errs = append(errs, fmt.Sprintf("%s: %v", url, err))
		}
	}
	if len(errs) == 0 {
		return ErrNoDownloadURLs
	}
	return fmt.Errorf("all downloads failed: %s", strings.Join(errs, "; "))
}

func Extract(ctx context.Context, src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, f)

	var files lewfs.Files
	if z, err := zipfs.Open(ctx, f); err == nil {
		files = lewfs.Walk(ctx, z, nil)
	} else if img, err := squashfs.Open(ctx, f); err == nil {
		files = lewfs.Walk(ctx, img, nil)
	} else if tfs, err := tarfs.Open(ctx, f); err == nil {
		files = lewfs.Walk(ctx, tfs, nil)
	} else if errors.Is(err, fs.ErrInvalid) {
		return err
	} else {
		return installBinary(ctx, src, dest)
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	root, err := lewpath.Open(dest)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, root)
	return lewfs.Copy(ctx, root, files)
}

func MoveContents(srcDir, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(destDir, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func downloadWithFetchurl(ctx context.Context, url, dest string, opts DownloadOptions) error {
	// Note: "name" (basename) used to be for a local progressWriter.
	// The central progressTransport in the httpclient now handles task naming
	// ("fetch:<basename>") and progress automatically.

	fetcher, err := driver.Get[fetchurl.Driver](ctx)
	if err != nil {
		return err
	}

	out, err := atomicfile.Create(dest, 0)
	if err != nil {
		return err
	}
	defer logging.RunCleanup(ctx, "atomicfile.Abort", out.Abort)

	// HTTP progress is owned by httpclient.WithProgress (one fetch bar).
	// Isolate so a failing verified fetchurl attempt does not cancel the parent
	// group before DownloadFile falls back to downloadDirect.
	algo, hash := parseHash(opts.Hash)
	fetchErr := taskgroup.Isolate(ctx, func(ctx context.Context) error {
		return fetcher.Fetch(ctx, fetchurl.FetchOptions{
			URLs: []string{url},
			Algo: algo,
			Hash: hash,
			Out:  out,
			Size: opts.Size,
		})
	})
	if fetchErr != nil {
		return fetchErr
	}
	return out.CommitMode(opts.Mode)
}

func downloadDirect(ctx context.Context, url, dest string, opts DownloadOptions) error {
	// Perform the actual GET + atomicfile.Create/Commit.
	// If the ctx carries a taskgroup (normal case under tool install / apply),
	// the httpclient driver's WithProgress transport will see the request and
	// promote *this* HTTP to exactly one Internet task (title = basename),
	// driving real progress via progressReadCloser on body reads.
	// This gives the asset download a single specific task (dispatch point for
	// either the fetchurl path or this direct path) without an extra wrapper
	// layer around it.
	//
	// Isolate around fetchurl attempts (in downloadWithFetchurl) keeps tasks
	// from a failing verified attempt from recording errors on the parent group,
	// so the fallback here can still succeed cleanly.

	out, err := atomicfile.Create(dest, 0)
	if err != nil {
		return err
	}
	defer logging.RunCleanup(ctx, "atomicfile.Abort", out.Abort)

	httpClient, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return fmt.Errorf("get http client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if opts.ConfigureRequest != nil {
		opts.ConfigureRequest(req)
	}

	resp, err := httpClient.Client().Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		logging.Close(ctx, resp.Body)
		err := fmt.Errorf("GET %s: %s", url, resp.Status)
		if resp.StatusCode == http.StatusForbidden {
			err = fmt.Errorf("%w (if this is a GitHub release asset, set GITHUB_TOKEN or run 'gh auth login' to increase rate limits)", err)
		}
		return err
	}

	// resp.Body is the original (no group) or a progressReadCloser (group present
	// at the time of this Do). io.Copy will drive the single transport-created
	// task's progress when applicable.
	_, copyErr := io.Copy(out, resp.Body)
	logging.Close(ctx, resp.Body)
	if copyErr != nil {
		return copyErr
	}
	return out.CommitMode(opts.Mode)
}

func parseHash(raw string) (algo, hash string) {
	algo, hash = "sha256", raw
	if a, h, ok := strings.Cut(raw, ":"); ok {
		algo, hash = a, h
	}
	return algo, hash
}

func installBinary(ctx context.Context, src, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer logging.Close(ctx, in)

	outPath := filepath.Join(dest, NormalizeBinaryName(filepath.Base(src)))
	return atomicfile.Write(outPath, in, 0o755)
}

var versionPattern = regexp.MustCompile(constants.BinaryVersionPattern)

func NormalizeBinaryName(name string) string {
	result := name
	for _, suffix := range constants.BinaryNameSuffixes {
		if before, ok := strings.CutSuffix(result, suffix); ok {
			result = before
			break
		}
	}
	return versionPattern.ReplaceAllString(result, "")
}

// NormalizeInstalledBinaries renames top-level and bin/ executables whose
// basenames still embed platform triples (e.g. codex-x86_64-unknown-linux-musl
// -> codex) using NormalizeBinaryName.
func NormalizeInstalledBinaries(destDir string) error {
	for _, dir := range []string{destDir, filepath.Join(destDir, "bin")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			oldName := entry.Name()
			newName := NormalizeBinaryName(oldName)
			if newName == "" || newName == oldName {
				continue
			}
			oldPath := filepath.Join(dir, oldName)
			info, err := os.Stat(oldPath)
			if err != nil {
				return err
			}
			if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
				continue
			}
			newPath := filepath.Join(dir, newName)
			if _, err := os.Stat(newPath); err == nil {
				continue
			}
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
		}
	}
	return nil
}
