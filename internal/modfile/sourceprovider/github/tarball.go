package github

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	lewfs "github.com/lewtec/lewkit/x/fs"
	tarfs "github.com/lewtec/lewkit/x/fs/tar"
	lewpath "github.com/lewtec/lewkit/x/path"

	"github.com/lucasew/workspaced/internal/archive"
	"github.com/lucasew/workspaced/internal/githubutil"
	"github.com/lucasew/workspaced/pkg/driver"
	httpclientdriver "github.com/lucasew/workspaced/pkg/driver/httpclient"
	"github.com/lucasew/workspaced/pkg/logging"
)

func downloadAndExtractTarball(ctx context.Context, source Source, destDir string, expectedHash string) (sourceMeta, error) {
	url, err := source.ResolvePinnedTarballURL(ctx)
	if err != nil {
		return sourceMeta{}, err
	}
	hash, err := fetchAndExtractTarballURL(ctx, url, destDir, expectedHash)
	if err != nil {
		return sourceMeta{}, err
	}
	return sourceMeta{
		URL:  url,
		Hash: hash,
	}, nil
}

// fetchAndExtractTarballURL downloads a GitHub tarball via httpclient with auth.
// We cannot use the fetchurl driver here: private repos need Authorization on the
// request, and fetchurl has no ConfigureRequest hook. Hash is verified locally
// when expectedHash is set.
func fetchAndExtractTarballURL(ctx context.Context, url string, destDir string, expectedHash string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", githubutil.UserAgent)
	githubutil.ApplyAuth(ctx, req)

	httpDriver, err := driver.Get[httpclientdriver.Driver](ctx)
	if err != nil {
		return "", fmt.Errorf("get http client driver: %w", err)
	}
	resp, err := httpDriver.Client().Do(req)
	if err != nil {
		return "", err
	}
	defer logging.Close(ctx, resp.Body)
	if resp.StatusCode != http.StatusOK {
		hint := ""
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
			if githubutil.Token(ctx) == "" {
				hint = " (private repos require GITHUB_TOKEN or 'gh auth login')"
			}
		}
		return "", fmt.Errorf("unexpected status: %s%s", resp.Status, hint)
	}

	h := sha256.New()
	body := io.TeeReader(resp.Body, h)
	root, err := lewpath.Open(destDir)
	if err != nil {
		return "", err
	}
	defer logging.Close(ctx, root)
	tfs, err := tarfs.Open(body)
	if err != nil {
		return "", err
	}
	if err := lewfs.Copy(ctx, root, lewfs.Walk(tfs, nil)); err != nil {
		return "", err
	}
	if err := archive.StripTopLevelDir(destDir); err != nil {
		return "", err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if expectedHash != "" && got != expectedHash {
		return "", fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, got)
	}
	return got, nil
}
