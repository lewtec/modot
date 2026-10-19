package modfile

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const tagSpecPrefix = "tag:"

var (
	ErrEmptyTagPattern      = errors.New("tag pattern is empty")
	ErrInvalidTagPattern    = errors.New("tag pattern is invalid")
	ErrNoMatchingTag        = errors.New("no tag matches pattern")
	ErrUnresolvedTagPattern = errors.New("unresolved tag pattern")
)

// TagSpec reports the regexp carried by a version of the form tag:<regexp>.
// The pattern is kept as written. Compilation translates (?<name> to Go's (?P<name>.
func TagSpec(ref string) (pattern string, ok bool, err error) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, tagSpecPrefix) {
		return "", false, nil
	}
	pattern = strings.TrimSpace(strings.TrimPrefix(ref, tagSpecPrefix))
	if pattern == "" {
		return "", false, ErrEmptyTagPattern
	}
	if _, err := compileTagPattern(pattern); err != nil {
		return "", false, fmt.Errorf("%w: %w", ErrInvalidTagPattern, err)
	}
	return pattern, true, nil
}

// TagVersioning is the Renovate versioning string for a tag pattern.
func TagVersioning(pattern string) string {
	return "regex:" + pattern
}

// TagMatches reports whether name matches a tag pattern.
func TagMatches(pattern, name string) bool {
	re, err := compileTagPattern(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(strings.TrimSpace(name))
}

// SelectNewestTag returns the newest tag whose name matches pattern.
// Ranking uses a version, major/minor/patch, or the full match, in that order.
// Numeric segments compare as integers. A prerelease is older than its release.
func SelectNewestTag(tags []string, pattern string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", ErrEmptyTagPattern
	}
	re, err := compileTagPattern(pattern)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidTagPattern, err)
	}
	var (
		best  string
		bestR versionRank
		found bool
	)
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || !re.MatchString(tag) {
			continue
		}
		rank, ok := rankTag(re, tag)
		if !ok {
			continue
		}
		if !found || rank.newerThan(bestR) {
			best = tag
			bestR = rank
			found = true
		}
	}
	if !found {
		return "", ErrNoMatchingTag
	}
	return best, nil
}

func compileTagPattern(pattern string) (*regexp.Regexp, error) {
	goPattern := strings.ReplaceAll(pattern, "(?<", "(?P<")
	return regexp.Compile(goPattern)
}

type versionRank struct {
	nums    []int
	pre     []string
	raw     string
	lexical bool
}

func rankTag(re *regexp.Regexp, tag string) (versionRank, bool) {
	m := re.FindStringSubmatch(tag)
	if m == nil {
		return versionRank{}, false
	}
	text := versionText(re, m)
	if strings.TrimSpace(text) == "" {
		return versionRank{}, false
	}
	return parseVersionRank(text), true
}

func versionText(re *regexp.Regexp, match []string) string {
	names := re.SubexpNames()
	version := ""
	major, minor, patch := "", "", ""
	hasParts := false
	for i, name := range names {
		if i == 0 || i >= len(match) {
			continue
		}
		switch name {
		case "version":
			version = match[i]
		case "major":
			major = match[i]
			hasParts = true
		case "minor":
			minor = match[i]
			hasParts = true
		case "patch":
			patch = match[i]
			hasParts = true
		}
	}
	if version != "" {
		return version
	}
	if hasParts {
		if major == "" {
			major = "0"
		}
		if minor == "" {
			minor = "0"
		}
		if patch == "" {
			patch = "0"
		}
		return major + "." + minor + "." + patch
	}
	if len(match) > 0 {
		return match[0]
	}
	return ""
}

func parseVersionRank(raw string) versionRank {
	raw = strings.TrimSpace(raw)
	if raw != "" && (raw[0] == 'v' || raw[0] == 'V') {
		raw = raw[1:]
	}
	main, pre, hasPre := strings.Cut(raw, "-")
	rank := versionRank{raw: raw}
	if hasPre {
		rank.pre = strings.Split(pre, ".")
	}
	if main == "" {
		rank.lexical = true
		return rank
	}
	for _, part := range strings.Split(main, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || part == "" {
			rank.lexical = true
			rank.nums = nil
			return rank
		}
		rank.nums = append(rank.nums, n)
	}
	return rank
}

func (a versionRank) newerThan(b versionRank) bool {
	if a.lexical || b.lexical {
		return a.raw > b.raw
	}
	n := len(a.nums)
	if len(b.nums) > n {
		n = len(b.nums)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(a.nums) {
			av = a.nums[i]
		}
		if i < len(b.nums) {
			bv = b.nums[i]
		}
		if av != bv {
			return av > bv
		}
	}
	if len(a.pre) == 0 && len(b.pre) > 0 {
		return true
	}
	if len(a.pre) > 0 && len(b.pre) == 0 {
		return false
	}
	pn := len(a.pre)
	if len(b.pre) > pn {
		pn = len(b.pre)
	}
	for i := 0; i < pn; i++ {
		if i >= len(a.pre) {
			return false
		}
		if i >= len(b.pre) {
			return true
		}
		switch cmpPreIdent(a.pre[i], b.pre[i]) {
		case 1:
			return true
		case -1:
			return false
		}
	}
	return false
}

func cmpPreIdent(a, b string) int {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	switch {
	case aerr == nil && berr == nil:
		if ai == bi {
			return 0
		}
		if ai > bi {
			return 1
		}
		return -1
	case aerr == nil:
		return -1
	case berr == nil:
		return 1
	default:
		return strings.Compare(a, b)
	}
}
