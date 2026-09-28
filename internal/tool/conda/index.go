package conda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/lewtec/lewkit/x/driver/fetchurl"
	"github.com/lewtec/lewkit/x/taskgroup"
	lewtool "github.com/lewtec/lewkit/x/tool"
)

// record is one installable file from a conda channel.
type record struct {
	Name        string
	Version     string
	Build       string
	BuildNumber int
	Depends     []string
	SHA256      string
	MD5         string
	Size        int64
	Subdir      string
	Filename    string
	URL         string
}

type fetcher func(ctx context.Context, rawURL string) ([]byte, error)

type index struct {
	fetch    fetcher
	mu       sync.Mutex
	cache    map[string][]record
	inflight map[string]*indexCall
}

// indexCall joins concurrent lookups of one package.
type indexCall struct {
	done chan struct{}
	recs []record
	err  error
}

func newIndex(fetch fetcher) *index {
	if fetch == nil {
		fetch = defaultFetch
	}
	return &index{
		fetch:    fetch,
		cache:    map[string][]record{},
		inflight: map[string]*indexCall{},
	}
}

func (ix *index) records(ctx context.Context, ch channel, name, subdir string) ([]record, error) {
	name = strings.ToLower(name)
	key := ch.Base + "\x00" + name
	ix.mu.Lock()
	if recs, ok := ix.cache[key]; ok {
		ix.mu.Unlock()
		return recs, nil
	}
	if call, ok := ix.inflight[key]; ok {
		ix.mu.Unlock()
		select {
		case <-call.done:
			return call.recs, call.err
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	}
	call := &indexCall{done: make(chan struct{})}
	ix.inflight[key] = call
	ix.mu.Unlock()

	recs, err := ix.load(ctx, ch, name, subdir)

	ix.mu.Lock()
	if err == nil {
		ix.cache[key] = recs
	}
	delete(ix.inflight, key)
	call.recs = recs
	call.err = err
	close(call.done)
	ix.mu.Unlock()
	return recs, err
}

func (ix *index) load(ctx context.Context, ch channel, name, subdir string) ([]record, error) {
	if apiURL, ok := ch.apiURL(name); ok {
		recs, err := ix.recordsFromAPI(ctx, apiURL, name)
		if err == nil {
			return recs, nil
		}
		if !errors.Is(err, ErrPackageNotFound) {
			return nil, err
		}
	}
	return ix.recordsFromRepodata(ctx, ch, name, subdir)
}

func (ix *index) recordsFromAPI(ctx context.Context, rawURL, name string) ([]record, error) {
	body, err := ix.fetch(ctx, rawURL)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrPackageNotFound, name)
		}
		return nil, err
	}
	var payload apiPackage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode %s: %w", rawURL, err)
	}
	files := preferMainLabel(payload.Files)
	recs := make([]record, 0, len(files))
	for _, file := range files {
		rec, ok := file.toRecord(name)
		if ok {
			recs = append(recs, rec)
		}
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrPackageNotFound, name)
	}
	return recs, nil
}

func (ix *index) recordsFromRepodata(ctx context.Context, ch channel, name, subdir string) ([]record, error) {
	if subdir == "" {
		resolved, err := currentSubdir()
		if err != nil {
			return nil, err
		}
		subdir = resolved
	}
	dirs := []string{subdir}
	if subdir != subdirNoarch {
		dirs = append(dirs, subdirNoarch)
	}
	bodies := make([][]byte, len(dirs))
	// Each subdir fetch is a Control task. DownloadFile takes an Internet
	// slot, and that pool is the backpressure.
	err := taskgroup.WithSession(ctx, func(ctx context.Context) error {
		_, runErr := taskgroup.Map[int, struct{}]{
			Name:     "conda:repodata",
			Items:    seq(len(dirs)),
			PoolKind: taskgroup.Control,
			TaskName: func(_ int, i int) string { return "repodata:" + dirs[i] },
			Fn: func(ctx context.Context, _ *taskgroup.Status, i int) (struct{}, error) {
				body, ferr := ix.fetchRepodata(ctx, ch, dirs[i])
				if ferr != nil {
					if isNotFound(ferr) {
						return struct{}{}, nil
					}
					return struct{}{}, ferr
				}
				bodies[i] = body
				return struct{}{}, nil
			},
		}.Run(ctx)
		return runErr
	})
	if err != nil {
		return nil, err
	}
	var recs []record
	foundIndex := false
	for i, body := range bodies {
		if body == nil {
			continue
		}
		foundIndex = true
		parsed, err := decodeRepodata(body)
		if err != nil {
			return nil, fmt.Errorf("decode repodata %s/%s: %w", ch.Base, dirs[i], err)
		}
		recs = append(recs, parsed.records(name, dirs[i])...)
	}
	if !foundIndex || len(recs) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrPackageNotFound, name)
	}
	return recs, nil
}

func seq(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func (ix *index) fetchRepodata(ctx context.Context, ch channel, subdir string) ([]byte, error) {
	body, err := ix.fetch(ctx, ch.repodataURL(subdir, true))
	if err == nil {
		return body, nil
	}
	if !isNotFound(err) {
		return nil, err
	}
	return ix.fetch(ctx, ch.repodataURL(subdir, false))
}

func isNotFound(err error) bool {
	var status *fetchurl.StatusError
	return errors.As(err, &status) && status.Code == http.StatusNotFound
}

type apiPackage struct {
	Files []apiFile `json:"files"`
}

type apiFile struct {
	Basename string   `json:"basename"`
	Version  string   `json:"version"`
	SHA256   string   `json:"sha256"`
	MD5      string   `json:"md5"`
	Size     int64    `json:"size"`
	Labels   []string `json:"labels"`
	Attrs    apiAttrs `json:"attrs"`
}

type apiAttrs struct {
	Build       string   `json:"build"`
	BuildNumber int      `json:"build_number"`
	Depends     []string `json:"depends"`
	Subdir      string   `json:"subdir"`
}

func preferMainLabel(files []apiFile) []apiFile {
	var main []apiFile
	for _, file := range files {
		if len(file.Labels) == 0 || containsLabel(file.Labels, "main") {
			main = append(main, file)
		}
	}
	if len(main) == 0 {
		return files
	}
	return main
}

func containsLabel(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}

func (file apiFile) toRecord(name string) (record, bool) {
	filename := path.Base(file.Basename)
	if filename == "" || filename == "." {
		return record{}, false
	}
	subdir := file.Attrs.Subdir
	if subdir == "" {
		dir := path.Dir(file.Basename)
		if dir != "." {
			subdir = dir
		}
	}
	if subdir == "" || file.Version == "" {
		return record{}, false
	}
	build := file.Attrs.Build
	if build == "" {
		build = buildFromFilename(name, file.Version, filename)
	}
	depends := file.Attrs.Depends
	if depends == nil {
		depends = []string{}
	}
	return record{
		Name:        name,
		Version:     file.Version,
		Build:       build,
		BuildNumber: file.Attrs.BuildNumber,
		Depends:     depends,
		SHA256:      file.SHA256,
		MD5:         file.MD5,
		Size:        file.Size,
		Subdir:      subdir,
		Filename:    filename,
	}, true
}

func buildFromFilename(name, version, filename string) string {
	stem := strings.TrimSuffix(strings.TrimSuffix(filename, ".conda"), ".tar.bz2")
	prefix := name + "-" + version + "-"
	if strings.HasPrefix(stem, prefix) {
		return strings.TrimPrefix(stem, prefix)
	}
	return ""
}

type repodataFile struct {
	Packages      map[string]repodataRecord `json:"packages"`
	PackagesConda map[string]repodataRecord `json:"packages.conda"`
}

type repodataRecord struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Build       string   `json:"build"`
	BuildNumber int      `json:"build_number"`
	Depends     []string `json:"depends"`
	SHA256      string   `json:"sha256"`
	MD5         string   `json:"md5"`
	Size        int64    `json:"size"`
	Subdir      string   `json:"subdir"`
}

func decodeRepodata(body []byte) (repodataFile, error) {
	payload := body
	if isZstd(body) {
		decoded, err := decodeZstd(body)
		if err != nil {
			return repodataFile{}, err
		}
		payload = decoded
	}
	var file repodataFile
	if err := json.Unmarshal(payload, &file); err != nil {
		return repodataFile{}, err
	}
	return file, nil
}

func isZstd(body []byte) bool {
	return len(body) >= 4 && body[0] == 0x28 && body[1] == 0xb5 && body[2] == 0x2f && body[3] == 0xfd
}

func decodeZstd(body []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	return io.ReadAll(decoder)
}

func (file repodataFile) records(name, subdir string) []record {
	var recs []record
	collect := func(filename string, item repodataRecord) {
		if !strings.EqualFold(item.Name, name) {
			return
		}
		dir := item.Subdir
		if dir == "" {
			dir = subdir
		}
		depends := item.Depends
		if depends == nil {
			depends = []string{}
		}
		recs = append(recs, record{
			Name:        strings.ToLower(item.Name),
			Version:     item.Version,
			Build:       item.Build,
			BuildNumber: item.BuildNumber,
			Depends:     depends,
			SHA256:      item.SHA256,
			MD5:         item.MD5,
			Size:        item.Size,
			Subdir:      dir,
			Filename:    path.Base(filename),
		})
	}
	for filename, item := range file.PackagesConda {
		collect(filename, item)
	}
	for filename, item := range file.Packages {
		collect(filename, item)
	}
	return recs
}

func defaultFetch(ctx context.Context, rawURL string) ([]byte, error) {
	file, err := os.CreateTemp("", "conda-index-*")
	if err != nil {
		return nil, err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		os.Remove(name)
		return nil, err
	}
	defer os.Remove(name)
	if err := lewtool.DownloadFile(ctx, rawURL, name, lewtool.DownloadOptions{}); err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	body, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rawURL, err)
	}
	return body, nil
}
