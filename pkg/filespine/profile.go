package filespine

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
	for _, profile := range Profiles {
		if profile == name {
			return true
		}
	}
	return false
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
		return profile != ModeCodebase && profile != ModeSystem
	}
}
