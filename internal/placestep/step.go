// Package placestep rewrites one origin tree before it is placed.
// core:place and file mounts share it. Steps run in name order.
// require checks gitignore patterns. move renames a path or directory prefix.
package placestep

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/git-pkgs/gitignore"
)

var (
	// ErrRequireNoMatch is a positive pattern that hit nothing.
	ErrRequireNoMatch = errors.New("no path matched")
	// ErrRequireMustNot is a "!pattern" that hit a path.
	ErrRequireMustNot = errors.New("pattern must not match")
	// ErrBadPattern is a gitignore pattern the matcher rejected.
	ErrBadPattern = errors.New("invalid pattern")
	// ErrEmptyPath is a move path that is blank or ".".
	ErrEmptyPath = errors.New("empty path")
	// ErrPathEscape is a move path that leaves the origin tree.
	ErrPathEscape = errors.New("path escapes origin")
	// ErrMoveFrom is a bad move source.
	ErrMoveFrom = errors.New("move from path")
	// ErrMoveTo is a bad move destination.
	ErrMoveTo = errors.New("move to path")
)

// Entry is one file in the origin tree. Rel is slash-separated and origin-relative.
type Entry struct {
	Rel     string
	Abs     string
	Mode    os.FileMode
	Symlink bool
}

// Step is one named operation. Unknown ops are skipped.
type Step struct {
	Op       string
	From     string
	To       string
	Patterns map[string]string
}

// Apply runs steps in name order. subject prefixes error and warning text.
func Apply(subject string, steps map[string]Step, entries []Entry) ([]Entry, []string, error) {
	names := make([]string, 0, len(steps))
	for name := range steps {
		names = append(names, name)
	}
	sort.Strings(names)

	var warnings []string
	var err error
	for _, name := range names {
		entries, warnings, err = run{
			subject: subject,
			name:    name,
			step:    steps[name],
		}.apply(entries, warnings)
		if err != nil {
			return nil, warnings, err
		}
	}
	return entries, warnings, nil
}

type run struct {
	subject string
	name    string
	step    Step
}

func (r run) apply(entries []Entry, warnings []string) ([]Entry, []string, error) {
	switch strings.TrimSpace(r.step.Op) {
	case "move":
		return r.move(entries, warnings)
	case "require":
		return r.require(entries, warnings)
	default:
		return entries, warnings, nil
	}
}

func (r run) move(entries []Entry, warnings []string) ([]Entry, []string, error) {
	from, err := Clean(r.step.From)
	if err != nil {
		return nil, warnings, fmt.Errorf("%s step %q: %w: %w", r.subject, r.name, ErrMoveFrom, err)
	}
	to, err := Clean(r.step.To)
	if err != nil {
		return nil, warnings, fmt.Errorf("%s step %q: %w: %w", r.subject, r.name, ErrMoveTo, err)
	}

	prefix := from + "/"
	var moving, staying []Entry
	for _, entry := range entries {
		if entry.Rel == from || strings.HasPrefix(entry.Rel, prefix) {
			moving = append(moving, entry)
			continue
		}
		staying = append(staying, entry)
	}
	if len(moving) == 0 {
		warnings = append(warnings, fmt.Sprintf("%s step %q: move source %q missing; skipped", r.subject, r.name, from))
		return entries, warnings, nil
	}

	byRel := make(map[string]Entry, len(staying)+len(moving))
	for _, entry := range staying {
		byRel[entry.Rel] = entry
	}
	for _, entry := range moving {
		newRel := Rewrite(entry.Rel, from, to)
		if prev, ok := byRel[newRel]; ok {
			warnings = append(warnings, fmt.Sprintf("%s step %q: move %q → %q overwrites existing %q", r.subject, r.name, entry.Rel, newRel, prev.Abs))
		}
		entry.Rel = newRel
		byRel[newRel] = entry
	}

	out := make([]Entry, 0, len(byRel))
	for _, entry := range byRel {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, warnings, nil
}

func (r run) require(entries []Entry, warnings []string) ([]Entry, []string, error) {
	names := make([]string, 0, len(r.step.Patterns))
	for name := range r.step.Patterns {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		raw := strings.TrimSpace(r.step.Patterns[name])
		if raw == "" {
			continue
		}
		negate := strings.HasPrefix(raw, "!")
		pat := raw
		if negate {
			pat = strings.TrimSpace(strings.TrimPrefix(raw, "!"))
			if pat == "" {
				continue
			}
		}

		matcher := gitignore.New("")
		matcher.AddPatterns([]byte(pat+"\n"), "")
		if errs := matcher.Errors(); len(errs) > 0 {
			return nil, warnings, fmt.Errorf("%s step %q pattern %q: %w: %v", r.subject, r.name, name, ErrBadPattern, errs[0])
		}

		matched := false
		for _, entry := range entries {
			if matcher.MatchPath(entry.Rel, false) {
				matched = true
				break
			}
		}
		if negate {
			if matched {
				return nil, warnings, fmt.Errorf("%s step %q pattern %q (%s): %w", r.subject, r.name, name, raw, ErrRequireMustNot)
			}
			continue
		}
		if !matched {
			return nil, warnings, fmt.Errorf("%s step %q pattern %q (%s): %w", r.subject, r.name, name, raw, ErrRequireNoMatch)
		}
	}
	return entries, warnings, nil
}

// Clean is a slash path inside the origin tree.
func Clean(p string) (string, error) {
	p = strings.TrimSpace(p)
	p = filepath.ToSlash(p)
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return "", ErrEmptyPath
	}
	for seg := range strings.SplitSeq(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("%w: %q", ErrPathEscape, p)
		}
	}
	cleaned := path.Clean(p)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: %q", ErrPathEscape, p)
	}
	return cleaned, nil
}

// Rewrite replaces the from prefix of rel with to.
func Rewrite(rel, from, to string) string {
	if rel == from {
		return to
	}
	rest := strings.TrimPrefix(rel, from+"/")
	if to == "" {
		return rest
	}
	return to + "/" + rest
}
