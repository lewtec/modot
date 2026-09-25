package media

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	kitmedia "github.com/lewtec/lewkit/x/driver/media"
	"github.com/lewtec/modot/internal/atomicfile"
	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/driver/httpclient"
	"github.com/lewtec/modot/internal/logging"
)

type PlaybackStatus = kitmedia.PlaybackStatus

const (
	StatusPlaying = kitmedia.StatusPlaying
	StatusPaused  = kitmedia.StatusPaused
	StatusStopped = kitmedia.StatusStopped
)

type Metadata = kitmedia.Metadata
type Driver = kitmedia.Driver

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

	httpDriver, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return "", err
	}

	resp, err := httpDriver.Client().Get(url)
	if err != nil {
		return "", err
	}
	defer logging.Close(ctx, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	if err := atomicfile.Write(path, resp.Body, 0); err != nil {
		return "", err
	}

	return path, nil
}
