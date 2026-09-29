package conda

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Version ordering follows conda.models.version.VersionOrder
// (fillvalue 0, so 1.1 == 1.1.0).

var (
	versionCheckRe = regexp.MustCompile(`^[\*\.\+!_0-9a-z]+$`)
	versionSplitRe = regexp.MustCompile(`([0-9]+|[*]+|[^0-9*]+)`)
)

type atomKind int

const (
	atomString atomKind = iota
	atomNumber
	atomPost
)

type atom struct {
	kind atomKind
	num  int
	str  string
}

var fillAtom = atom{kind: atomNumber, num: 0}

func (a atom) equal(b atom) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case atomNumber:
		return a.num == b.num
	case atomString:
		return a.str == b.str
	case atomPost:
		return true
	default:
		return false
	}
}

// less reports a < b. Strings sort before numbers, and post sorts after both.
func (a atom) less(b atom) bool {
	if a.equal(b) {
		return false
	}
	if rank(a) != rank(b) {
		return rank(a) < rank(b)
	}
	switch a.kind {
	case atomString:
		return a.str < b.str
	case atomNumber:
		return a.num < b.num
	default:
		return false
	}
}

func rank(a atom) int {
	switch a.kind {
	case atomString:
		return 0
	case atomNumber:
		return 1
	case atomPost:
		return 2
	default:
		return 1
	}
}

type condaVersion struct {
	original string
	parts    [][]atom
	local    [][]atom
}

func parseCondaVersion(raw string) (condaVersion, error) {
	original := strings.TrimSpace(raw)
	version := strings.ToLower(original)
	if version == "" {
		return condaVersion{}, fmt.Errorf("%w: empty version", ErrInvalidVersion)
	}
	if !versionCheckRe.MatchString(version) {
		if strings.Contains(version, "-") && !strings.Contains(version, "_") {
			version = strings.ReplaceAll(version, "-", "_")
		}
		if !versionCheckRe.MatchString(version) {
			return condaVersion{}, fmt.Errorf("%w: %q", ErrInvalidVersion, raw)
		}
	}

	epoch, rest, err := splitEpoch(version)
	if err != nil {
		return condaVersion{}, err
	}
	main, local, err := splitLocal(rest)
	if err != nil {
		return condaVersion{}, err
	}
	parts, err := splitVersion(main)
	if err != nil {
		return condaVersion{}, err
	}
	localParts, err := splitComponents(local)
	if err != nil {
		return condaVersion{}, err
	}
	return condaVersion{
		original: version,
		parts:    append([][]atom{{epoch}}, parts...),
		local:    localParts,
	}, nil
}

func splitEpoch(version string) (atom, string, error) {
	pieces := strings.Split(version, "!")
	switch len(pieces) {
	case 1:
		return atom{kind: atomNumber, num: 0}, pieces[0], nil
	case 2:
		if pieces[0] == "" || strings.IndexFunc(pieces[0], unicode.IsDigit) != 0 || !allDigits(pieces[0]) {
			return atom{}, "", fmt.Errorf("%w: epoch must be an integer", ErrInvalidVersion)
		}
		number := 0
		for _, r := range pieces[0] {
			number = number*10 + int(r-'0')
		}
		return atom{kind: atomNumber, num: number}, pieces[1], nil
	default:
		return atom{}, "", fmt.Errorf("%w: duplicated epoch separator", ErrInvalidVersion)
	}
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func splitLocal(version string) (string, []string, error) {
	pieces := strings.Split(version, "+")
	switch len(pieces) {
	case 1:
		return pieces[0], nil, nil
	case 2:
		if pieces[0] == "" {
			return "", nil, fmt.Errorf("%w: missing version before '+'", ErrInvalidVersion)
		}
		local := strings.Split(strings.ReplaceAll(pieces[1], "_", "."), ".")
		return pieces[0], local, nil
	default:
		return "", nil, fmt.Errorf("%w: duplicated local version separator", ErrInvalidVersion)
	}
}

func splitVersion(version string) ([][]atom, error) {
	if version == "" {
		return nil, fmt.Errorf("%w: empty version", ErrInvalidVersion)
	}
	var segments []string
	if strings.HasSuffix(version, "_") {
		segments = strings.Split(strings.ReplaceAll(version[:len(version)-1], "_", "."), ".")
		if len(segments) == 0 || segments[0] == "" {
			return nil, fmt.Errorf("%w: empty version component", ErrInvalidVersion)
		}
		segments[len(segments)-1] += "_"
	} else {
		segments = strings.Split(strings.ReplaceAll(version, "_", "."), ".")
	}
	return splitComponents(segments)
}

func splitComponents(segments []string) ([][]atom, error) {
	out := make([][]atom, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			return nil, fmt.Errorf("%w: empty version component", ErrInvalidVersion)
		}
		pieces := versionSplitRe.FindAllString(segment, -1)
		if len(pieces) == 0 {
			return nil, fmt.Errorf("%w: empty version component", ErrInvalidVersion)
		}
		atoms := make([]atom, 0, len(pieces)+1)
		for _, piece := range pieces {
			atoms = append(atoms, parseAtom(piece))
		}
		if segment[0] < '0' || segment[0] > '9' {
			atoms = append([]atom{fillAtom}, atoms...)
		}
		out = append(out, atoms)
	}
	return out, nil
}

func parseAtom(piece string) atom {
	switch piece {
	case "post":
		return atom{kind: atomPost}
	case "dev":
		return atom{kind: atomString, str: "DEV"}
	}
	if allDigits(piece) {
		number := 0
		for _, r := range piece {
			number = number*10 + int(r-'0')
		}
		return atom{kind: atomNumber, num: number}
	}
	return atom{kind: atomString, str: piece}
}

func compareVersions(left, right string) (int, error) {
	a, err := parseCondaVersion(left)
	if err != nil {
		return 0, err
	}
	b, err := parseCondaVersion(right)
	if err != nil {
		return 0, err
	}
	return a.compare(b), nil
}

func (v condaVersion) compare(other condaVersion) int {
	if cmp := compareParts(v.parts, other.parts); cmp != 0 {
		return cmp
	}
	return compareParts(v.local, other.local)
}

func (v condaVersion) equal(other condaVersion) bool {
	return v.compare(other) == 0
}

func compareParts(left, right [][]atom) int {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	for i := 0; i < n; i++ {
		var a, b []atom
		if i < len(left) {
			a = left[i]
		}
		if i < len(right) {
			b = right[i]
		}
		m := len(a)
		if len(b) > m {
			m = len(b)
		}
		for j := 0; j < m; j++ {
			c1 := fillAtom
			c2 := fillAtom
			if j < len(a) {
				c1 = a[j]
			}
			if j < len(b) {
				c2 = b[j]
			}
			if c1.equal(c2) {
				continue
			}
			if c1.less(c2) {
				return -1
			}
			return 1
		}
	}
	return 0
}

// hasPrefix reports whether v starts with other, using conda's VersionOrder.startswith.
func (v condaVersion) hasPrefix(other condaVersion) bool {
	left := v.parts
	right := other.parts
	if len(other.local) > 0 {
		if !partsEqual(v.parts, other.parts) {
			return false
		}
		left = v.local
		right = other.local
	}
	if len(right) == 0 {
		return false
	}
	cut := len(right) - 1
	if !partsEqual(prefixParts(left, cut), prefixParts(right, cut)) {
		return false
	}
	var head []atom
	if len(left) > cut {
		head = left[cut]
	}
	tail := right[cut]
	inner := len(tail) - 1
	if inner < 0 {
		return false
	}
	if !componentPrefixEqual(head, tail, inner) {
		return false
	}
	c1 := fillAtom
	if len(head) > inner {
		c1 = head[inner]
	}
	c2 := tail[inner]
	if c2.kind == atomString {
		return c1.kind == atomString && strings.HasPrefix(c1.str, c2.str)
	}
	return c1.equal(c2)
}

func prefixParts(parts [][]atom, n int) [][]atom {
	if n <= 0 {
		return nil
	}
	if n > len(parts) {
		return parts
	}
	return parts[:n]
}

func partsEqual(left, right [][]atom) bool {
	return compareParts(left, right) == 0
}

func componentPrefixEqual(left, right []atom, n int) bool {
	if n <= 0 {
		return true
	}
	l := left
	r := right
	if n < len(l) {
		l = l[:n]
	}
	if n < len(r) {
		r = r[:n]
	}
	return compareParts([][]atom{l}, [][]atom{r}) == 0
}
