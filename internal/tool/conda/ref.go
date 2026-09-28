package conda

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	defaultChannelName = "conda-forge"
	anacondaOrg        = "https://conda.anaconda.org/"
	anacondaAPI        = "https://api.anaconda.org/package/"
)

// channel is one conda repository. Base is the channel root without a
// trailing slash. APIOwner is the anaconda.org owner when that index can
// answer a single package without downloading repodata.
type channel struct {
	Name     string
	Base     string
	APIOwner string
}

type parsedRef struct {
	Channel channel
	Name    string
}

var knownChannels = map[string]channel{
	"conda-forge": {
		Name:     "conda-forge",
		Base:     "https://conda.anaconda.org/conda-forge",
		APIOwner: "conda-forge",
	},
	"bioconda": {
		Name:     "bioconda",
		Base:     "https://conda.anaconda.org/bioconda",
		APIOwner: "bioconda",
	},
	"defaults": {
		Name:     "defaults",
		Base:     "https://repo.anaconda.com/pkgs/main",
		APIOwner: "main",
	},
	"main": {
		Name:     "main",
		Base:     "https://repo.anaconda.com/pkgs/main",
		APIOwner: "main",
	},
	"anaconda": {
		Name:     "anaconda",
		Base:     "https://repo.anaconda.com/pkgs/main",
		APIOwner: "anaconda",
	},
	"pkgs/main": {
		Name:     "pkgs/main",
		Base:     "https://repo.anaconda.com/pkgs/main",
		APIOwner: "main",
	},
}

// parseRef reads a conda package ref.
//
//	ripgrep
//	conda-forge/ripgrep
//	bioconda::samtools
//	https://conda.anaconda.org/conda-forge::ripgrep
//
// A bare name uses the conda-forge channel. The last slash-separated
// segment is the package name. Channel paths such as pkgs/main/curl keep
// every segment except the last.
func parseRef(ref string) (parsedRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return parsedRef{}, ErrEmptyRef
	}

	channelName := defaultChannelName
	name := ref
	if before, after, ok := strings.Cut(ref, "::"); ok {
		channelName = strings.TrimSpace(before)
		name = strings.TrimSpace(after)
	} else if strings.Contains(ref, "/") && !strings.Contains(ref, "://") {
		slash := strings.LastIndex(ref, "/")
		channelName = strings.TrimSpace(ref[:slash])
		name = strings.TrimSpace(ref[slash+1:])
	}
	if channelName == "" || name == "" || strings.Contains(name, "/") {
		return parsedRef{}, fmt.Errorf("%w: %q", ErrInvalidRef, ref)
	}

	ch, err := resolveChannel(channelName)
	if err != nil {
		return parsedRef{}, err
	}
	return parsedRef{Channel: ch, Name: strings.ToLower(name)}, nil
}

func resolveChannel(name string) (channel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultChannelName
	}
	if strings.Contains(name, "://") {
		parsed, err := url.Parse(name)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return channel{}, fmt.Errorf("%w: channel %q", ErrInvalidRef, name)
		}
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		return channel{Name: name, Base: parsed.String()}, nil
	}

	key := strings.ToLower(strings.Trim(name, "/"))
	if ch, ok := knownChannels[key]; ok {
		return ch, nil
	}
	owner := ""
	if !strings.Contains(key, "/") {
		owner = key
	}
	return channel{
		Name:     key,
		Base:     strings.TrimRight(anacondaOrg, "/") + "/" + key,
		APIOwner: owner,
	}, nil
}

func (ch channel) apiURL(name string) (string, bool) {
	if ch.APIOwner == "" || name == "" {
		return "", false
	}
	return anacondaAPI + url.PathEscape(ch.APIOwner) + "/" + url.PathEscape(name), true
}

func (ch channel) artifactURL(subdir, filename string) string {
	return strings.TrimRight(ch.Base, "/") + "/" + subdir + "/" + filename
}

func (ch channel) repodataURL(subdir string, zstd bool) string {
	name := "repodata.json"
	if zstd {
		name = "repodata.json.zst"
	}
	return strings.TrimRight(ch.Base, "/") + "/" + subdir + "/" + name
}
