package filespine

import (
	"slices"

	lewpath "github.com/lewtec/lewkit/x/path"
)

const (
	ModeHome     = "home"
	ModeCodebase = "codebase"
	ModeSystem   = "system"
)

// Profiles are the only legal keys under file.
// etc, usr, var, bin, and root are paths inside system, not profiles.
var Profiles = []string{
	ModeHome,
	ModeCodebase,
	ModeSystem,
}

// IsNamespace reports whether name is a dest profile.
func IsNamespace(name string) bool {
	return slices.Contains(Profiles, name)
}

// Visible profiles for mode, in Profiles order.
func Visible(mode string) []string {
	out := make([]string, 0, len(Profiles))
	for _, profile := range Profiles {
		if NamespaceVisible(mode, profile) {
			out = append(out, profile)
		}
	}
	return out
}

// PresetVisible reports whether a module directory is read for mode.
// System mode reads etc, usr, var, bin, root, and system into one tree.
func PresetVisible(mode, preset string) bool {
	switch mode {
	case ModeCodebase:
		return preset == ModeCodebase
	case ModeSystem:
		return SystemRel(preset) != "" || preset == ModeSystem || preset == "root"
	default:
		return preset == ModeHome
	}
}

// SystemRel is the path of a system preset inside the system root.
// root and system are the root itself. An unknown name returns "".
func SystemRel(preset string) string {
	switch preset {
	case "etc", "usr", "var":
		return preset
	case "bin":
		return "usr/local/bin"
	case "root", ModeSystem:
		return "."
	default:
		return ""
	}
}

// ApplyDir joins a profile onto root.
// An empty root is ".". home, codebase, and system are the root itself.
func ApplyDir(profile, root string) string {
	if root == "" {
		root = "."
	}
	if profile != "" && !IsNamespace(profile) {
		if rel := SystemRel(profile); rel != "" && rel != "." {
			return HostPath(root, lewpath.New(rel))
		}
	}
	return root
}

// ProfileForTarget reports the profile whose apply directory is target.
// An empty target, or target equal to primary, selects Primary(mode).
func ProfileForTarget(mode, target, primary string) (string, bool) {
	if target == "" || target == primary || target == "." {
		return Primary(mode), true
	}
	for _, profile := range Visible(mode) {
		if ApplyDir(profile, primary) == target {
			return profile, true
		}
	}
	return "", false
}

// Primary is the profile that receives files aimed at the apply target.
// An empty mode is home.
func Primary(mode string) string {
	switch mode {
	case ModeCodebase:
		return ModeCodebase
	case ModeSystem:
		return ModeSystem
	default:
		return ModeHome
	}
}

// NamespaceVisible reports whether profile is emitted for mode.
// An empty mode is home.
func NamespaceVisible(mode, profile string) bool {
	switch mode {
	case ModeCodebase:
		return profile == ModeCodebase
	case ModeSystem:
		return profile == ModeSystem
	default:
		return profile == ModeHome
	}
}
