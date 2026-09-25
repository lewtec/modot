package modot

// Common prelude injected into every modot.cue evaluation.
inputs: self: {
	from: *"self" | string
}
// Pins live in the workspace lockfile, not here. Product code must
// resolve these via lazy_tools (ResolveLazyTool / needs), never a
// hardcoded spec@version.
lazy_tools: {
	mise: {
		ref:  *"registry:mise" | string
		bins: *["mise"] | [...string]
	}
	gh: {
		ref:  *"github:cli/cli" | string
		bins: *["gh"] | [...string]
	}
	resvg: {
		ref:  *"registry:resvg" | string
		bins: *["resvg"] | [...string]
	}
}
drivers: {
	"github.com/lewtec/modot/internal/driver/clipboard.Driver": {
		"clipboard_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/dialog.Chooser": {
		"terminal": *0 | int
	}
	"github.com/lewtec/modot/internal/driver/dialog.Confirmer": {
		"terminal": *0 | int
	}
	"github.com/lewtec/modot/internal/driver/dialog.Prompter": {
		"terminal": *0 | int
	}
	"github.com/lewtec/modot/internal/driver/env.Driver": {
		"env_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/exec.Driver": {
		"exec_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/httpclient.Driver": {
		"httpclient_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/notification.Driver": {
		"notification_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/opener.Driver": {
		"opener_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/power.Driver": {
		"power_termux": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/rsync.Driver": {
		"rsync_native": *60 | int
	}
	"github.com/lewtec/modot/internal/driver/svgraster.Driver": {
		"resvg": *100 | int
	}
	"github.com/lewtec/modot/internal/driver/terminal.Driver": {
		"terminal_termux": *60 | int
	}
}
