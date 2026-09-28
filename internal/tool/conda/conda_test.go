package conda

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/lewtec/lewkit/x/driver/fetchurl"
	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/stretchr/testify/require"
)

func TestParseRef(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		channel string
		base    string
		name    string
	}{
		{in: "ripgrep", channel: "conda-forge", base: "https://conda.anaconda.org/conda-forge", name: "ripgrep"},
		{in: "bioconda/samtools", channel: "bioconda", base: "https://conda.anaconda.org/bioconda", name: "samtools"},
		{in: "defaults::curl", channel: "defaults", base: "https://repo.anaconda.com/pkgs/main", name: "curl"},
		{in: "pkgs/main/curl", channel: "pkgs/main", base: "https://repo.anaconda.com/pkgs/main", name: "curl"},
		{in: "conda-forge::Ripgrep", channel: "conda-forge", base: "https://conda.anaconda.org/conda-forge", name: "ripgrep"},
		{in: "https://example.test/my-channel::pkg", channel: "https://example.test/my-channel", base: "https://example.test/my-channel", name: "pkg"},
	}
	for _, tc := range cases {
		parsed, err := parseRef(tc.in)
		require.NoError(t, err, tc.in)
		require.Equal(t, tc.channel, parsed.Channel.Name, tc.in)
		require.Equal(t, tc.base, parsed.Channel.Base, tc.in)
		require.Equal(t, tc.name, parsed.Name, tc.in)
	}
	_, err := parseRef("  ")
	require.ErrorIs(t, err, ErrEmptyRef)
}

func TestMatchSpec(t *testing.T) {
	t.Parallel()
	spec, err := parseMatchSpec("libcurl 7.80.0 h0b77cf5_0")
	require.NoError(t, err)
	require.Equal(t, "libcurl", spec.Name)
	require.Equal(t, "7.80.0", spec.Version)
	require.Equal(t, "h0b77cf5_0", spec.Build)

	spec, err = parseMatchSpec("numpy=1.11")
	require.NoError(t, err)
	require.Equal(t, "numpy", spec.Name)
	require.Equal(t, "=1.11", spec.Version)

	spec, err = parseMatchSpec("libgcc >=15")
	require.NoError(t, err)
	require.Equal(t, ">=15", spec.Version)
	require.Empty(t, spec.Build)

	spec, err = parseMatchSpec("defaults::curl >=8")
	require.NoError(t, err)
	require.Equal(t, "defaults", spec.Channel)
	require.Equal(t, "curl", spec.Name)
}

func TestHostSubdir(t *testing.T) {
	t.Parallel()
	subdir, err := hostSubdir("linux", "amd64")
	require.NoError(t, err)
	require.Equal(t, "linux-64", subdir)
	subdir, err = hostSubdir("darwin", "arm64")
	require.NoError(t, err)
	require.Equal(t, "osx-arm64", subdir)
	_, err = hostSubdir("plan9", "amd64")
	require.ErrorIs(t, err, ErrNoBuild)
}

func TestBackendRegistered(t *testing.T) {
	backend, err := lewtool.Get("conda")
	require.NoError(t, err)
	require.Equal(t, "conda", backend.Name())
	_, err = lewtool.Get("missing-backend")
	require.ErrorIs(t, err, lewtool.ErrBackendNotFound)
}

func TestListVersionsAndInstall(t *testing.T) {
	t.Parallel()
	hello := condaPackage(t, map[string]archiveFile{
		"bin/hello": {body: "#!/opt/anaconda1anaconda2anaconda3/bin/sh\necho hello\n", mode: 0o755},
	}, []prefixFile{{
		Path:        "bin/hello",
		Mode:        "text",
		Placeholder: "/opt/anaconda1anaconda2anaconda3",
	}})
	lib := condaPackage(t, map[string]archiveFile{
		"lib/libhello.txt": {body: "lib", mode: 0o644},
	}, nil)

	const (
		helloAPI = "https://api.anaconda.org/package/conda-forge/hello"
		libAPI   = "https://api.anaconda.org/package/conda-forge/libhello"
	)
	bodies := map[string][]byte{
		helloAPI: apiJSON(t, "hello", []apiFile{
			apiEntry("hello", "1.0.0", "h0", 0, "linux-64", []string{"libhello >=1"}, []string{"main"}),
			apiEntry("hello", "1.2.0a1", "h1", 1, "linux-64", []string{"libhello >=1"}, []string{"main"}),
			apiEntry("hello", "1.2.0", "h2", 2, "linux-64", []string{"libhello >=1", "__glibc >=2.17"}, []string{"main"}),
			apiEntry("hello", "1.2.0", "h9", 9, "win-64", nil, []string{"main"}),
			apiEntry("hello", "9.9.9", "h0", 0, "linux-64", nil, []string{"broken"}),
		}),
		libAPI: apiJSON(t, "libhello", []apiFile{
			apiEntry("libhello", "1.0.0", "h0", 0, "linux-64", nil, []string{"main"}),
			apiEntry("libhello", "1.1.0", "h1", 1, "linux-64", nil, []string{"main"}),
		}),
	}
	downloads := map[string]string{
		"https://conda.anaconda.org/conda-forge/linux-64/hello-1.2.0-h2.conda":    hello,
		"https://conda.anaconda.org/conda-forge/linux-64/libhello-1.1.0-h1.conda": lib,
	}
	backend := &Backend{
		subdir: "linux-64",
		fetch: func(_ context.Context, rawURL string) ([]byte, error) {
			body, ok := bodies[rawURL]
			if !ok {
				return nil, &fetchurl.StatusError{Code: 404, Status: "404"}
			}
			return body, nil
		},
		download: func(_ context.Context, rawURL, dest, _ string, _ int64) error {
			src, ok := downloads[rawURL]
			if !ok {
				return &fetchurl.StatusError{Code: 404, Status: "404 " + rawURL}
			}
			payload, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			return os.WriteFile(dest, payload, 0o644)
		},
	}

	tool, err := backend.Tool("hello")
	require.NoError(t, err)
	versions, err := tool.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"1.2.0", "1.2.0a1", "1.0.0"}, versions)

	pin := tool.(lewtool.Pinner).Pin()
	require.Equal(t, "conda-forge/hello", pin.Name)
	require.Equal(t, "conda", pin.Datasource)

	dest := t.TempDir()
	require.NoError(t, tool.Install(t.Context(), "1.2.0", dest))
	script, err := os.ReadFile(filepath.Join(dest, "bin", "hello"))
	require.NoError(t, err)
	require.Contains(t, string(script), "#!"+dest+"/bin/sh")
	require.NotContains(t, string(script), defaultPlaceholder)
	libBody, err := os.ReadFile(filepath.Join(dest, "lib", "libhello.txt"))
	require.NoError(t, err)
	require.Equal(t, "lib", string(libBody))
	info, err := os.Stat(filepath.Join(dest, "bin", "hello"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111)
}

func TestRepodataFallback(t *testing.T) {
	t.Parallel()
	channelBase := "https://example.test/chan"
	repodata := repodataFile{
		PackagesConda: map[string]repodataRecord{
			"jq-1.7.1-h1.conda": {
				Name: "jq", Version: "1.7.1", Build: "h1", BuildNumber: 1,
				SHA256: "abc", Subdir: "linux-64",
			},
		},
		Packages: map[string]repodataRecord{
			"jq-1.6-h0.tar.bz2": {
				Name: "jq", Version: "1.6", Build: "h0", BuildNumber: 0, Subdir: "linux-64",
			},
		},
	}
	payload, err := json.Marshal(repodata)
	require.NoError(t, err)
	ix := newIndex(func(_ context.Context, rawURL string) ([]byte, error) {
		if rawURL == channelBase+"/linux-64/repodata.json" {
			return payload, nil
		}
		return nil, &fetchurl.StatusError{Code: 404, Status: "404"}
	})
	recs, err := ix.records(t.Context(), channel{Name: "custom", Base: channelBase}, "jq", "linux-64")
	require.NoError(t, err)
	picked, err := pickRecord(recs, "", "", "linux-64")
	require.NoError(t, err)
	require.Equal(t, "1.7.1", picked.Version)
	require.Equal(t, "jq-1.7.1-h1.conda", picked.Filename)
}

func TestClosureOrdersSharedDependency(t *testing.T) {
	t.Parallel()
	bodies := map[string][]byte{
		"https://api.anaconda.org/package/conda-forge/top": apiJSON(t, "top", []apiFile{
			apiEntry("top", "1.0.0", "h0", 0, "linux-64", []string{"left", "right"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/left": apiJSON(t, "left", []apiFile{
			apiEntry("left", "1.0.0", "h0", 0, "linux-64", []string{"common"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/right": apiJSON(t, "right", []apiFile{
			apiEntry("right", "1.0.0", "h0", 0, "linux-64", []string{"common"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/common": apiJSON(t, "common", []apiFile{
			apiEntry("common", "1.0.0", "h0", 0, "linux-64", nil, []string{"main"}),
		}),
	}
	ix := newIndex(func(_ context.Context, rawURL string) ([]byte, error) {
		body, ok := bodies[rawURL]
		if !ok {
			return nil, &fetchurl.StatusError{Code: 404, Status: "404"}
		}
		return body, nil
	})
	recs, err := ix.closure(t.Context(), knownChannels["conda-forge"], "top", "", "", "linux-64", nil)
	require.NoError(t, err)
	pos := map[string]int{}
	for i, rec := range recs {
		pos[rec.Name] = i
	}
	require.Less(t, pos["common"], pos["left"])
	require.Less(t, pos["common"], pos["right"])
	require.Less(t, pos["left"], pos["top"])
	require.Less(t, pos["right"], pos["top"])
}

func TestSplitPackageBuildsStayPaired(t *testing.T) {
	t.Parallel()
	// libmeta has two builds that pin different libfoo builds. A loose
	// libfoo request is solved at the same time and would otherwise keep
	// the other build.
	bodies := map[string][]byte{
		"https://api.anaconda.org/package/conda-forge/app": apiJSON(t, "app", []apiFile{
			apiEntry("app", "1.0.0", "h0", 0, "linux-64", []string{"libmeta", "libfoo >=1"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/libmeta": apiJSON(t, "libmeta", []apiFile{
			apiEntry("libmeta", "1.0.0", "a_pin_b", 0, "linux-64", []string{"libfoo 1.0.0 build_b"}, []string{"main"}),
			apiEntry("libmeta", "1.0.0", "b_pin_a", 0, "linux-64", []string{"libfoo 1.0.0 build_a"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/libfoo": apiJSON(t, "libfoo", []apiFile{
			apiEntry("libfoo", "1.0.0", "build_a", 0, "linux-64", nil, []string{"main"}),
			apiEntry("libfoo", "1.0.0", "build_b", 0, "linux-64", []string{"icu >=1"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/icu": apiJSON(t, "icu", []apiFile{
			apiEntry("icu", "1.0.0", "h0", 0, "linux-64", nil, []string{"main"}),
		}),
	}
	ix := newIndex(func(_ context.Context, rawURL string) ([]byte, error) {
		body, ok := bodies[rawURL]
		if !ok {
			return nil, &fetchurl.StatusError{Code: 404, Status: "404"}
		}
		return body, nil
	})
	recs, err := ix.closure(t.Context(), knownChannels["conda-forge"], "app", "", "", "linux-64", nil)
	require.NoError(t, err)
	got := map[string]string{}
	for _, rec := range recs {
		got[rec.Name] = rec.Build
	}
	// Filename order selects libmeta a_pin_b, which requires libfoo build_b.
	require.Equal(t, "a_pin_b", got["libmeta"])
	require.Equal(t, "build_b", got["libfoo"])
	require.Equal(t, "h0", got["icu"])
	pos := map[string]int{}
	for i, rec := range recs {
		pos[rec.Name] = i
	}
	require.Less(t, pos["icu"], pos["libfoo"])
	require.Less(t, pos["libfoo"], pos["libmeta"])
}

func TestFetchAndDownloadOverlap(t *testing.T) {
	t.Parallel()
	hello := condaPackage(t, map[string]archiveFile{
		"bin/hello": {body: "hello\n", mode: 0o755},
	}, nil)
	lib := condaPackage(t, map[string]archiveFile{
		"lib/libhello.txt": {body: "lib", mode: 0o644},
	}, nil)
	const (
		helloAPI = "https://api.anaconda.org/package/conda-forge/hello"
		libAPI   = "https://api.anaconda.org/package/conda-forge/libhello"
	)
	bodies := map[string][]byte{
		helloAPI: apiJSON(t, "hello", []apiFile{
			apiEntry("hello", "1.0.0", "h0", 0, "linux-64", []string{"libhello >=1"}, []string{"main"}),
		}),
		libAPI: apiJSON(t, "libhello", []apiFile{
			apiEntry("libhello", "1.0.0", "h0", 0, "linux-64", nil, []string{"main"}),
		}),
	}
	fetchLib := make(chan struct{})
	downloadHello := make(chan struct{})
	backend := &Backend{
		subdir: "linux-64",
		fetch: func(ctx context.Context, rawURL string) ([]byte, error) {
			body, ok := bodies[rawURL]
			if !ok {
				return nil, &fetchurl.StatusError{Code: 404, Status: "404"}
			}
			if rawURL != libAPI {
				return body, nil
			}
			close(fetchLib)
			select {
			case <-downloadHello:
				return body, nil
			case <-ctx.Done():
				return nil, context.Cause(ctx)
			}
		},
		download: func(ctx context.Context, rawURL, dest, _ string, _ int64) error {
			src := lib
			if strings.Contains(rawURL, "/hello-") {
				src = hello
				close(downloadHello)
				select {
				case <-fetchLib:
				case <-ctx.Done():
					return context.Cause(ctx)
				}
			}
			payload, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			return os.WriteFile(dest, payload, 0o644)
		},
	}
	tool, err := backend.Tool("hello")
	require.NoError(t, err)
	require.NoError(t, tool.Install(t.Context(), "1.0.0", t.TempDir()))
}

func TestDependencyConflict(t *testing.T) {
	t.Parallel()
	bodies := map[string][]byte{
		"https://api.anaconda.org/package/conda-forge/left": apiJSON(t, "left", []apiFile{
			apiEntry("left", "1.0.0", "h0", 0, "linux-64", []string{"common >=2"}, []string{"main"}),
		}),
		"https://api.anaconda.org/package/conda-forge/common": apiJSON(t, "common", []apiFile{
			apiEntry("common", "1.0.0", "h0", 0, "linux-64", []string{"left >=1,<2"}, []string{"main"}),
		}),
	}
	// left >=2 is unsatisfied because common is only 1.0.0, and the cycle
	// also asks left for <2. The first failure is the missing common build.
	ix := newIndex(func(_ context.Context, rawURL string) ([]byte, error) {
		body, ok := bodies[rawURL]
		if !ok {
			return nil, &fetchurl.StatusError{Code: 404, Status: "404"}
		}
		return body, nil
	})
	_, err := ix.closure(t.Context(), knownChannels["conda-forge"], "left", "", "", "linux-64", nil)
	require.ErrorIs(t, err, ErrUnsatisfied)
}

func TestPrefixPlaceholderStringForm(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"paths":[
		{"_path":"include/zlib.h","path_type":"hardlink"},
		{"_path":"lib/pkgconfig/zlib.pc","path_type":"hardlink","file_mode":"text","prefix_placeholder":"/opt/anaconda1anaconda2anaconda3"},
		{"_path":"bin/tool","path_type":"hardlink","prefix_placeholder":{"file_mode":"binary","placeholder":"PLACE"}}
	]}`)
	files, err := prefixFromPaths(payload)
	require.NoError(t, err)
	require.Equal(t, []prefixFile{
		{Path: "lib/pkgconfig/zlib.pc", Type: "hardlink", Mode: "text", Placeholder: "/opt/anaconda1anaconda2anaconda3"},
		{Path: "bin/tool", Type: "hardlink", Mode: "binary", Placeholder: "PLACE"},
	}, files)
}

func TestPrefixBinaryPadding(t *testing.T) {
	t.Parallel()
	// The placeholder begins a C string. The suffix through the NUL stays,
	// and the bytes saved by a shorter prefix are NUL padding after it.
	data, err := replacePlaceholder([]byte("SEARCH_DIR(\"=PLACE___/lib\");\x00NEXT"), "PLACE___", "/tmp", "binary")
	require.NoError(t, err)
	require.Equal(t, "SEARCH_DIR(\"=/tmp/lib\");\x00\x00\x00\x00\x00NEXT", string(data))
	_, err = replacePlaceholder([]byte("PLACE\x00"), "PLACE", "too-long-prefix", "binary")
	require.ErrorIs(t, err, ErrPrefixTooLong)
}

func TestArchivePathEscape(t *testing.T) {
	t.Parallel()
	_, err := safeRel("../etc/passwd")
	require.ErrorIs(t, err, ErrPathEscapes)
	rel, err := safeRel("./bin/rg")
	require.NoError(t, err)
	require.Equal(t, "bin/rg", rel)
}

func TestInstallPrefixStripsStagingUUID(t *testing.T) {
	t.Parallel()
	got := installPrefix("/tmp/tools/conda-ripgrep/15.2.0.018f6d3e-7c31-7a1b-8d4e-9c0b1a2d3e4f")
	require.Equal(t, "/tmp/tools/conda-ripgrep/15.2.0", got)
	require.Equal(t, "/tmp/tools/conda-ripgrep/15.2.0", installPrefix("/tmp/tools/conda-ripgrep/15.2.0"))
}

type archiveFile struct {
	body string
	mode int64
}

func condaPackage(t *testing.T, files map[string]archiveFile, placeholders []prefixFile) string {
	t.Helper()
	pkg := tarZst(t, files)
	infoFiles := map[string]archiveFile{
		"info/paths.json": {body: pathsJSON(t, placeholders), mode: 0o644},
	}
	info := tarZst(t, infoFiles)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeZip(t, zw, "metadata.json", []byte(`{"conda_pkg_format_version":2}`))
	writeZip(t, zw, "pkg-test.tar.zst", pkg)
	writeZip(t, zw, "info-test.tar.zst", info)
	require.NoError(t, zw.Close())

	path := filepath.Join(t.TempDir(), "pkg.conda")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
	return path
}

func tarZst(t *testing.T, files map[string]archiveFile) []byte {
	t.Helper()
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	for name, file := range files {
		body := []byte(file.body)
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: file.mode,
			Size: int64(len(body)),
		}))
		_, err := tw.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())

	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = encoder.Write(raw.Bytes())
	require.NoError(t, err)
	require.NoError(t, encoder.Close())
	return compressed.Bytes()
}

func writeZip(t *testing.T, zw *zip.Writer, name string, payload []byte) {
	t.Helper()
	entry, err := zw.Create(name)
	require.NoError(t, err)
	_, err = entry.Write(payload)
	require.NoError(t, err)
}

func pathsJSON(t *testing.T, files []prefixFile) string {
	t.Helper()
	type item struct {
		Path              string `json:"_path"`
		Type              string `json:"path_type"`
		PrefixPlaceholder *struct {
			FileMode    string `json:"file_mode"`
			Placeholder string `json:"placeholder"`
		} `json:"prefix_placeholder,omitempty"`
	}
	doc := struct {
		Paths []item `json:"paths"`
	}{}
	for _, file := range files {
		entry := item{Path: file.Path, Type: "hardlink"}
		if file.Placeholder != "" {
			entry.PrefixPlaceholder = &struct {
				FileMode    string `json:"file_mode"`
				Placeholder string `json:"placeholder"`
			}{FileMode: file.Mode, Placeholder: file.Placeholder}
		}
		doc.Paths = append(doc.Paths, entry)
	}
	payload, err := json.Marshal(doc)
	require.NoError(t, err)
	return string(payload)
}

func apiJSON(t *testing.T, _ string, files []apiFile) []byte {
	t.Helper()
	payload, err := json.Marshal(apiPackage{Files: files})
	require.NoError(t, err)
	return payload
}

func apiEntry(name, version, build string, buildNumber int, subdir string, depends, labels []string) apiFile {
	return apiFile{
		Basename: subdir + "/" + name + "-" + version + "-" + build + ".conda",
		Version:  version,
		SHA256:   "abc",
		Size:     10,
		Labels:   labels,
		Attrs: apiAttrs{
			Build:       build,
			BuildNumber: buildNumber,
			Depends:     depends,
			Subdir:      subdir,
		},
	}
}
