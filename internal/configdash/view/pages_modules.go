// Package view renders the config dashboard pages.
package view

import (
	"encoding/json"
	"fmt"

	"github.com/lewtec/modot/internal/configdash"
)

// selectedModule is the module named on the page, or the first one.
func (v View) selectedModule() (configdash.Module, bool) {
	return selectedModule(v.shown(), v.ModuleName)
}

// tabBase is a daisyUI tab; the selected one is the active tab.
const tabBase = "tab"

// sourceLabel is the module's location the way the config names it:
// "core:base16-icons-linux" for a built-in, "self:modules/theme" for a
// directory in this workspace.
func sourceLabel(mod configdash.Module) string {
	if mod.Source == "" || mod.Ref == "" {
		return mod.Source + mod.Ref
	}

	return mod.Source + ":" + mod.Ref
}

// problemClass is the alert colour: a module that failed to load is an error,
// a built-in module with no file here is just information.
func problemClass(mod configdash.Module) string {
	if mod.Unavailable || mod.Problem == nil {
		return "alert alert-error items-start"
	}

	return "alert alert-info items-start"
}

// tabClass marks the selected module and the ones that failed to load, so a
// broken module stands out in the list without opening it.
func tabClass(mod configdash.Module, selected bool) string {
	classes := tabBase
	if selected {
		classes += " tab-active"
	}
	switch {
	case mod.Unavailable:
		// It failed to load: that is the thing to look at first.
		classes += " text-error"
	case mod.Problem != nil:
		// A built-in with no fragment here; explained, not broken.
		classes += " text-info"
	}

	return classes
}

func prettyJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}

	return string(encoded)
}
