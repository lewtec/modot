package cmdarg

import "github.com/lucasew/workspaced/pkg/palette/api"

// LintFormat is codebase lint --format.
type LintFormat int

const (
	LintTable LintFormat = iota + 1
	LintSARIF
)

func (f LintFormat) String() string {
	switch f {
	case LintSARIF:
		return "sarif"
	default:
		return "table"
	}
}

func (LintFormat) Values() []LintFormat {
	return []LintFormat{LintTable, LintSARIF}
}

// LayersFormat is config layers --format.
type LayersFormat int

const (
	LayersPaths LayersFormat = iota + 1
	LayersTable
)

func (f LayersFormat) String() string {
	switch f {
	case LayersTable:
		return "table"
	default:
		return "paths"
	}
}

func (LayersFormat) Values() []LayersFormat {
	return []LayersFormat{LayersPaths, LayersTable}
}

// Urgency is notification --urgency.
type Urgency int

const (
	UrgencyLow Urgency = iota + 1
	UrgencyNormal
	UrgencyCritical
)

func (u Urgency) String() string {
	switch u {
	case UrgencyLow:
		return "low"
	case UrgencyCritical:
		return "critical"
	default:
		return "normal"
	}
}

func (Urgency) Values() []Urgency {
	return []Urgency{UrgencyLow, UrgencyNormal, UrgencyCritical}
}

// Polarity is palette generate --polarity.
type Polarity int

const (
	PolarityAny Polarity = iota + 1
	PolarityDark
	PolarityLight
)

func (p Polarity) String() string {
	switch p {
	case PolarityDark:
		return "dark"
	case PolarityLight:
		return "light"
	default:
		return "any"
	}
}

func (Polarity) Values() []Polarity {
	return []Polarity{PolarityAny, PolarityDark, PolarityLight}
}

// API maps to palette api.Polarity.
func (p Polarity) API() api.Polarity {
	switch p {
	case PolarityDark:
		return api.PolarityDark
	case PolarityLight:
		return api.PolarityLight
	default:
		return api.PolarityAny
	}
}

// NixAction is utils nix deploy --action.
type NixAction int

const (
	NixActionSwitch NixAction = iota + 1
	NixActionBoot
	NixActionTest
)

func (a NixAction) String() string {
	switch a {
	case NixActionBoot:
		return "boot"
	case NixActionTest:
		return "test"
	default:
		return "switch"
	}
}

func (NixAction) Values() []NixAction {
	return []NixAction{NixActionSwitch, NixActionBoot, NixActionTest}
}

// HistorySource is utils history ingest.
type HistorySource int

const (
	HistoryBash HistorySource = iota + 1
	HistoryAtuin
)

func (s HistorySource) String() string {
	switch s {
	case HistoryAtuin:
		return "atuin"
	default:
		return "bash"
	}
}

func (HistorySource) Values() []HistorySource {
	return []HistorySource{HistoryBash, HistoryAtuin}
}
