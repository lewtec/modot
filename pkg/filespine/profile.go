package filespine

import "slices"

const (
	ModeHome     = "home"
	ModeCodebase = "codebase"
	ModeSystem   = "system"
)

// Profiles are the only legal keys under workspaced.file.
var Profiles = []string{
	"home",
	"codebase",
	"etc",
	"usr",
	"root",
	"var",
	"bin",
	"system",
}

// NamespaceBase is the apply directory for a fixed profile.
// home, codebase, and system use the apply target.
var NamespaceBase = map[string]string{
	"etc":  "/etc",
	"usr":  "/usr",
	"root": "/",
	"var":  "/var",
	"bin":  "/usr/local/bin",
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

// ApplyDir is where profile is written.
// home, codebase, and system use primary. The other profiles use NamespaceBase.
func ApplyDir(profile, primary string) string {
	if base, ok := NamespaceBase[profile]; ok {
		return base
	}
	return primary
}

// ProfileForTarget reports the profile whose apply directory is target.
// An empty target, or target equal to primary, selects Primary(mode).
func ProfileForTarget(mode, target, primary string) (string, bool) {
	if target == "" || target == primary {
		return Primary(mode), true
	}
	for _, profile := range Visible(mode) {
		if NamespaceBase[profile] == target {
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
// An empty mode is home. etc, usr, root, var, and bin are system.
func NamespaceVisible(mode, profile string) bool {
	switch mode {
	case ModeCodebase:
		return profile == ModeCodebase
	case ModeSystem:
		return IsNamespace(profile) && profile != ModeHome && profile != ModeCodebase
	default:
		return profile == ModeHome
	}
}
