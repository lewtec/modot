package conda

import (
	"fmt"
	"regexp"
	"strings"
)

// matchSpec is one conda dependency constraint: name, version, and build.
type matchSpec struct {
	Channel string
	Name    string
	Version string
	Build   string
}

type versionExpr interface {
	match(condaVersion) bool
}

type alwaysExpr struct{}

func (alwaysExpr) match(condaVersion) bool { return true }

type compareExpr struct {
	op  string
	ver condaVersion
}

func (e compareExpr) match(v condaVersion) bool {
	switch e.op {
	case "==":
		return v.equal(e.ver)
	case "!=":
		return !v.equal(e.ver)
	case "<":
		return v.compare(e.ver) < 0
	case "<=":
		return v.compare(e.ver) <= 0
	case ">":
		return v.compare(e.ver) > 0
	case ">=":
		return v.compare(e.ver) >= 0
	case "=":
		return v.hasPrefix(e.ver)
	case "!=startswith":
		return !v.hasPrefix(e.ver)
	case "~=":
		return compatibleRelease(v, e.ver)
	default:
		return false
	}
}

type andExpr struct{ parts []versionExpr }

func (e andExpr) match(v condaVersion) bool {
	for _, part := range e.parts {
		if !part.match(v) {
			return false
		}
	}
	return true
}

type orExpr struct{ parts []versionExpr }

func (e orExpr) match(v condaVersion) bool {
	for _, part := range e.parts {
		if part.match(v) {
			return true
		}
	}
	return false
}

type regexExpr struct{ re *regexp.Regexp }

func (e regexExpr) match(v condaVersion) bool {
	return e.re.MatchString(v.original)
}

func compatibleRelease(v, bound condaVersion) bool {
	pieces := strings.Split(bound.original, ".")
	if len(pieces) == 0 {
		return false
	}
	prefix, err := parseCondaVersion(strings.Join(pieces[:len(pieces)-1], "."))
	if err != nil {
		return false
	}
	return v.compare(bound) >= 0 && v.hasPrefix(prefix)
}

// parseMatchSpec splits "name [version [build]]", including channel::name
// and numpy=1.11 forms.
func parseMatchSpec(spec string) (matchSpec, error) {
	spec = strings.TrimSpace(spec)
	if cut := strings.IndexByte(spec, '['); cut >= 0 {
		spec = strings.TrimSpace(spec[:cut])
	}
	if spec == "" {
		return matchSpec{}, fmt.Errorf("%w: empty match spec", ErrInvalidRef)
	}

	channel := ""
	if before, after, ok := strings.Cut(spec, "::"); ok {
		channel = strings.TrimSpace(before)
		spec = strings.TrimSpace(after)
	}

	name, rest := splitName(spec)
	if name == "" {
		return matchSpec{}, fmt.Errorf("%w: %q", ErrInvalidRef, spec)
	}
	version, build := splitVersionBuild(rest)
	return matchSpec{Channel: channel, Name: strings.ToLower(name), Version: version, Build: build}, nil
}

func splitName(spec string) (string, string) {
	for i, r := range spec {
		if r == ' ' || r == '\t' || strings.ContainsRune("<>!=~", r) {
			return spec[:i], strings.TrimSpace(spec[i:])
		}
	}
	return spec, ""
}

func splitVersionBuild(rest string) (string, string) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", ""
	}
	fields := strings.Fields(rest)
	if len(fields) == 1 {
		return fields[0], ""
	}
	last := fields[len(fields)-1]
	if startsVersionToken(last) {
		return strings.Join(fields, ""), ""
	}
	return strings.Join(fields[:len(fields)-1], ""), last
}

func startsVersionToken(token string) bool {
	if token == "" {
		return false
	}
	return strings.ContainsRune("<>!=~|,()", rune(token[0])) || strings.HasPrefix(token, "==")
}

func matchVersion(spec, version string) (bool, error) {
	if strings.TrimSpace(spec) == "" || spec == "*" {
		return true, nil
	}
	expr, err := parseVersionExpr(spec)
	if err != nil {
		return false, err
	}
	parsed, err := parseCondaVersion(version)
	if err != nil {
		return false, err
	}
	return expr.match(parsed), nil
}

func matchBuild(spec, build string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "*" {
		return true
	}
	if strings.Contains(spec, "*") {
		ok, err := pathMatch(spec, build)
		return err == nil && ok
	}
	return spec == build
}

// pathMatch is path.Match without treating '\' as an escape, so windows
// paths are irrelevant and a build token like "py*" stays a glob.
func pathMatch(pattern, value string) (bool, error) {
	return regexp.MatchString("^"+globToRegexp(pattern)+"$", value)
}

func globToRegexp(pattern string) string {
	var b strings.Builder
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '.', '+', '?', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parseVersionExpr(spec string) (versionExpr, error) {
	tokens := tokenizeVersion(spec)
	expr, rest, err := parseOr(tokens)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("%w: %q", ErrInvalidVersion, spec)
	}
	return expr, nil
}

func tokenizeVersion(spec string) []string {
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tokens = append(tokens, b.String())
		b.Reset()
	}
	for _, r := range spec {
		switch r {
		case ' ', '\t':
			continue
		case '|', ',', '(', ')':
			flush()
			tokens = append(tokens, string(r))
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return tokens
}

func parseOr(tokens []string) (versionExpr, []string, error) {
	left, rest, err := parseAnd(tokens)
	if err != nil {
		return nil, nil, err
	}
	if len(rest) == 0 || rest[0] != "|" {
		return left, rest, nil
	}
	parts := []versionExpr{left}
	for len(rest) > 0 && rest[0] == "|" {
		var next versionExpr
		next, rest, err = parseAnd(rest[1:])
		if err != nil {
			return nil, nil, err
		}
		parts = append(parts, next)
	}
	return orExpr{parts: parts}, rest, nil
}

func parseAnd(tokens []string) (versionExpr, []string, error) {
	left, rest, err := parseFactor(tokens)
	if err != nil {
		return nil, nil, err
	}
	if len(rest) == 0 || rest[0] != "," {
		return left, rest, nil
	}
	parts := []versionExpr{left}
	for len(rest) > 0 && rest[0] == "," {
		var next versionExpr
		next, rest, err = parseFactor(rest[1:])
		if err != nil {
			return nil, nil, err
		}
		parts = append(parts, next)
	}
	return andExpr{parts: parts}, rest, nil
}

func parseFactor(tokens []string) (versionExpr, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("%w: empty version spec", ErrInvalidVersion)
	}
	if tokens[0] == "(" {
		expr, rest, err := parseOr(tokens[1:])
		if err != nil {
			return nil, nil, err
		}
		if len(rest) == 0 || rest[0] != ")" {
			return nil, nil, fmt.Errorf("%w: unclosed '('", ErrInvalidVersion)
		}
		return expr, rest[1:], nil
	}
	expr, err := parseVersionAtom(tokens[0])
	if err != nil {
		return nil, nil, err
	}
	return expr, tokens[1:], nil
}

func parseVersionAtom(token string) (versionExpr, error) {
	if token == "*" {
		return alwaysExpr{}, nil
	}
	if strings.HasPrefix(token, "^") || strings.HasSuffix(token, "$") {
		if !strings.HasPrefix(token, "^") || !strings.HasSuffix(token, "$") {
			return nil, fmt.Errorf("%w: regex spec must start with ^ and end with $", ErrInvalidVersion)
		}
		re, err := regexp.Compile(token)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidVersion, err)
		}
		return regexExpr{re: re}, nil
	}
	if op, rest, ok := cutOperator(token); ok {
		op, rest = normalizeDotStar(op, rest)
		ver, err := parseCondaVersion(rest)
		if err != nil {
			return nil, err
		}
		return compareExpr{op: op, ver: ver}, nil
	}
	if strings.Contains(token, "*") && !strings.HasSuffix(token, "*") {
		re, err := regexp.Compile("^" + globToRegexp(token) + "$")
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidVersion, err)
		}
		return regexExpr{re: re}, nil
	}
	if strings.HasSuffix(token, "*") {
		base := strings.TrimSuffix(token, "*")
		base = strings.TrimSuffix(base, ".")
		ver, err := parseCondaVersion(base)
		if err != nil {
			return nil, err
		}
		return compareExpr{op: "=", ver: ver}, nil
	}
	ver, err := parseCondaVersion(token)
	if err != nil {
		return nil, err
	}
	return compareExpr{op: "==", ver: ver}, nil
}

func cutOperator(token string) (string, string, bool) {
	for _, op := range []string{"==", "!=", "<=", ">=", "~=", "=", "<", ">"} {
		if rest, ok := strings.CutPrefix(token, op); ok && rest != "" && !strings.ContainsRune("=<>!~", rune(rest[0])) {
			return op, rest, true
		}
	}
	return "", token, false
}

func normalizeDotStar(op, rest string) (string, string) {
	if !strings.HasSuffix(rest, ".*") {
		return op, rest
	}
	rest = strings.TrimSuffix(rest, ".*")
	if op == "!=" {
		return "!=startswith", rest
	}
	return op, rest
}
