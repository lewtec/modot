package modfile

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/modot/internal/configcue"
)

var (
	ErrInvalidSourceSpec   = errors.New("expected provider:target[@ref]")
	ErrUnsupportedProvider = errors.New("source provider not supported for lock hash")
	ErrEmptyHash           = errors.New("provider returned empty hash")
	ErrInvalidLockEntry    = errors.New("invalid lock entry")
	ErrInputNotConfigured  = errors.New("module references input without config")
	ErrUnknownInput        = errors.New("unknown input")
	ErrInputMissingFrom    = errors.New("input is missing from field")
	ErrLockEntryShrunk     = errors.New("refusing to shrink source lock entries")
)

type SourceConfig struct {
	Provider   string `json:"provider"`
	Path       string `json:"path"`
	Repo       string `json:"repo"`
	URL        string `json:"url"`
	Ref        string `json:"ref"`
	Versioning string `json:"versioning,omitempty"`
}

type ModFile struct {
	Sources map[string]SourceConfig `json:"sources"`
}

type ResolvedModuleSource struct {
	Provider string
	Ref      string
	Version  string
}

func LoadModFile(path string) (*ModFile, error) {
	_ = path
	return &ModFile{Sources: map[string]SourceConfig{}}, nil
}

func ModFileFromConfig(cfg *configcue.Config) (*ModFile, error) {
	out := &ModFile{Sources: map[string]SourceConfig{}}
	if cfg == nil {
		return out, nil
	}
	inputs, err := cfg.Inputs()
	if err != nil {
		return out, nil
	}
	for name, input := range inputs {
		spec := strings.TrimSpace(input.From)
		if spec == "" {
			continue
		}
		if spec == "self" {
			out.Sources[name] = SourceConfig{
				Provider: "self",
				Ref:      strings.TrimSpace(input.Version),
			}
			continue
		}
		src, err := ParseSourceSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid input %q from %q: %w", name, spec, err)
		}
		src.Ref = strings.TrimSpace(input.Version)
		out.Sources[name] = src
	}

	// Always ensure a "self" source exists. This makes the canonical self
	// input directly referenceable in input refs (e.g. core:place items
	// can use "self:subdir" or "self:." without extra declarations).
	if _, ok := out.Sources["self"]; !ok {
		out.Sources["self"] = SourceConfig{Provider: "self"}
	}
	return out, nil
}

func (m *ModFile) ResolveModuleSource(moduleName, explicitFrom, modulesBaseDir string, sumFile *SumFile) (ResolvedModuleSource, error) {
	spec := strings.TrimSpace(explicitFrom)
	if spec == "" {
		spec = "self:modules/" + moduleName
	}

	left, right, ok := strings.Cut(spec, ":")
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if !ok || left == "" || right == "" {
		return ResolvedModuleSource{}, fmt.Errorf("invalid module source %q (expected <source-or-provider>:<path>[@version])", spec)
	}

	// Direct provider specs (no alias entry required): builtins + registered providers.
	if isDirectModuleSourcePrefix(left) {
		ref, version := SplitRefAndVersion(right)
		resolved, err := applyVersionLock(moduleName, left, ref, version, sumFile)
		if err != nil {
			return ResolvedModuleSource{}, err
		}
		return resolved, validateNonVersionedProvider(resolved)
	}

	// Alias source from modot inputs.
	src, ok := m.Sources[left]
	if !ok {
		return ResolvedModuleSource{}, fmt.Errorf("unknown source alias %q for module %q", left, moduleName)
	}
	src = applySourceLockOverlay(left, src, sumFile)
	if err := validateSourceLock(left, src, sumFile); err != nil {
		return ResolvedModuleSource{}, err
	}
	provider := strings.TrimSpace(src.Provider)
	if provider == "" {
		provider = left
	}

	if provider == "self" || provider == "local" {
		// Source inputs still say provider "local"; module provider ID is "self".
		// Both mean "filesystem path relative to workspace (or absolute)".
		base := filepath.Dir(modulesBaseDir)
		customBase := strings.TrimSpace(src.Path)
		if customBase != "" {
			if filepath.IsAbs(customBase) {
				base = customBase
			} else {
				base = filepath.Join(base, customBase)
			}
		} else if provider == "local" {
			// Bare local alias with no path: treat modulesBaseDir as the root.
			base = modulesBaseDir
		}
		ref, version := SplitRefAndVersion(right)
		resolved, err := applyVersionLock(moduleName, "self", filepath.Join(base, ref), version, sumFile)
		if err != nil {
			return ResolvedModuleSource{}, err
		}
		return resolved, validateNonVersionedProvider(resolved)
	}
	if p, ok := getSourceProvider(provider); ok {
		if fullRef, ver, handled, err := p.ResolveModuleRef(src, right); handled {
			if err != nil {
				return ResolvedModuleSource{}, fmt.Errorf("source alias %q (%s): %w", left, provider, err)
			}
			return applyVersionLock(moduleName, p.ID(), fullRef, ver, sumFile)
		}
	}
	ref, version := SplitRefAndVersion(right)
	return applyVersionLock(moduleName, provider, ref, version, sumFile)
}

// SplitRefAndVersion splits "ref@version" on the last '@'.
// If there is no usable version suffix, version is empty and ref is the trimmed input.
func SplitRefAndVersion(input string) (ref, version string) {
	in := strings.TrimSpace(input)
	idx := strings.LastIndex(in, "@")
	if idx <= 0 || idx == len(in)-1 {
		return in, ""
	}
	return strings.TrimSpace(in[:idx]), strings.TrimSpace(in[idx+1:])
}

func applyVersionLock(moduleName, provider, ref, version string, sumFile *SumFile) (ResolvedModuleSource, error) {
	resolved := ResolvedModuleSource{
		Provider: strings.TrimSpace(provider),
		Ref:      strings.TrimSpace(ref),
		Version:  strings.TrimSpace(version),
	}
	_ = moduleName
	_ = sumFile
	return resolved, nil
}

func validateNonVersionedProvider(source ResolvedModuleSource) error {
	if source.Version == "" {
		return nil
	}
	if source.Provider == "self" || source.Provider == "core" {
		return fmt.Errorf("provider %q does not support version pins", source.Provider)
	}
	return nil
}

func validateSourceLock(alias string, src SourceConfig, sumFile *SumFile) error {
	if sumFile == nil {
		return nil
	}
	lock, ok := sumFile.FindSource(alias)
	if !ok {
		return nil
	}
	provider := strings.TrimSpace(src.Provider)
	if provider == "" {
		provider = alias
	}
	if p, ok := getSourceProvider(provider); ok {
		src = p.Normalize(src)
	}
	if strings.TrimSpace(lock.Provider) != provider ||
		strings.TrimSpace(lock.Path) != strings.TrimSpace(src.Path) ||
		strings.TrimSpace(lock.Repo) != strings.TrimSpace(src.Repo) ||
		strings.TrimSpace(lock.URL) != strings.TrimSpace(src.URL) {
		return fmt.Errorf("source %q lock mismatch: run `modot mod lock`", alias)
	}
	return nil
}

// applySourceLockOverlay fills a ref from the lock.
// An empty version takes that row's tag or branch and its digest URL.
// A version tag:<regexp> takes the locked tag when the regexp matches, and drops the digest URL.
// Any other ref written in cue, including HEAD, stays as written.
func applySourceLockOverlay(alias string, src SourceConfig, sumFile *SumFile) SourceConfig {
	if sumFile == nil {
		return src
	}
	pattern, isTag, err := TagSpec(src.Ref)
	if err != nil {
		return src
	}
	if isTag {
		return overlayTagSpec(alias, src, sumFile, pattern)
	}
	if strings.TrimSpace(src.Ref) != "" {
		return src
	}
	lock, ok := lookupOverlayLock(alias, src, sumFile)
	if !ok {
		return src
	}
	if ref := strings.TrimSpace(lock.Ref); ref != "" && !strings.EqualFold(ref, "HEAD") {
		src.Ref = ref
	}
	if url := strings.TrimSpace(lock.URL); url != "" {
		src.URL = url
	}
	return src
}

func overlayTagSpec(alias string, src SourceConfig, sumFile *SumFile, pattern string) SourceConfig {
	lock, ok := lookupOverlayLock(alias, src, sumFile)
	if !ok {
		return src
	}
	ref := strings.TrimSpace(lock.Ref)
	if ref == "" || strings.EqualFold(ref, "HEAD") {
		return src
	}
	if _, lockedSpec, err := TagSpec(ref); lockedSpec || err != nil || !TagMatches(pattern, ref) {
		return src
	}
	src.Ref = ref
	src.URL = ""
	return src
}

func lookupOverlayLock(alias string, src SourceConfig, sumFile *SumFile) (LockedSource, bool) {
	lock, ok := findExistingSourceLock(sumFile, alias, lockedSourceIdentity(alias, src))
	if ok {
		return lock, true
	}
	repo := strings.TrimPrefix(strings.TrimSpace(src.Repo), "github:")
	if repo == "" {
		return LockedSource{}, false
	}
	if lock, ok = sumFile.FindSource("github:" + repo); ok {
		return lock, true
	}
	return sumFile.FindSource(repo)
}

func lockedSourceIdentity(alias string, src SourceConfig) LockedSource {
	provider := strings.TrimSpace(src.Provider)
	if provider == "" {
		provider = strings.TrimSpace(alias)
	}
	return LockedSource{
		Provider: provider,
		Path:     strings.TrimSpace(src.Path),
		Repo:     strings.TrimSpace(src.Repo),
		Ref:      strings.TrimSpace(src.Ref),
		URL:      strings.TrimSpace(src.URL),
	}
}

func ParseSourceSpec(spec string) (SourceConfig, error) {
	trimmed := strings.TrimSpace(spec)
	if !strings.Contains(trimmed, ":") {
		return SourceConfig{}, ErrInvalidSourceSpec
	}

	ts, err := lewtool.Parse(trimmed)
	if err != nil {
		return SourceConfig{}, err
	}
	provider := strings.TrimSpace(ts.Backend)
	target := strings.TrimSpace(ts.Package)
	ref := strings.TrimSpace(ts.Version)

	cfg := SourceConfig{Provider: provider, Ref: ref}
	if !strings.Contains(trimmed, "@") {
		cfg.Ref = ""
	}
	if p, ok := getSourceProvider(provider); ok {
		p.ConfigureFromSpec(&cfg, target)
	} else {
		cfg.Path = target
	}
	return cfg, nil
}
