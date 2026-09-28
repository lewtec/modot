package media

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	lewhttp "github.com/lewtec/lewkit/x/driver/httpclient"
	lewmedia "github.com/lewtec/lewkit/x/driver/media"
	"github.com/lewtec/modot/internal/atomicfile"
)

type PlaybackStatus = lewmedia.PlaybackStatus

const (
	StatusPlaying = lewmedia.StatusPlaying
	StatusPaused  = lewmedia.StatusPaused
	StatusStopped = lewmedia.StatusStopped
)

type Metadata = lewmedia.Metadata
type Driver = lewmedia.Driver

func GetArtCachePath(ctx context.Context, url string) (string, error) {
	if after, ok := strings.CutPrefix(url, "file://"); ok {
		return after, nil
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return url, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache/modot/media_art")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	path := filepath.Join(cacheDir, hash)

	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	httpDriver, err := lewdriver.Get[lewhttp.Driver](ctx)
	if err != nil {
		return "", err
	}

	resp, err := httpDriver.Client().Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if closer := resp.Body; closer != nil {
			if err := closer.Close(); err != nil {
				slog.Error("unexpected error", "op", "close", "error", err)
			}
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	if err := atomicfile.Write(path, resp.Body, 0); err != nil {
		return "", err
	}

	return path, nil
}
