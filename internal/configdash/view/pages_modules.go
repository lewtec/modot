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

// tabClass marks the selected module and the ones that failed to load, so a
// broken module stands out in the list without opening it.
func tabClass(mod configdash.Module, selected bool) string {
	classes := "tab"
	if selected {
		classes += " tab-active"
	}
	if mod.Problem != nil {
		classes += " text-error"
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
