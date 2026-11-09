package view

import (
	"encoding/json"
	"fmt"

	"github.com/lewtec/modot/internal/configdash"
)

// Markup is pre-rendered JSON for the merged options panel.
func (v View) selectedModule() (configdash.Module, bool) {
	return selectedModule(v.Report, v.ModuleName)
}

func prettyJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}
