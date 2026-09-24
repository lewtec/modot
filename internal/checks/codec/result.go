package codec

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/lucasew/workspaced/internal/checks"
	"github.com/owenrumney/go-sarif/v2/sarif"
)

// decodeJSONArray parses a JSON array of tool issues.
// Blank input and an empty array both yield a nil slice and a nil error.
func decodeJSONArray[T any](data []byte, what string) ([]T, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse %s output: %w", what, err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items, nil
}

// newNamedRun builds a SARIF run whose driver name falls back when toolName is empty.
func newNamedRun(toolName, fallback, infoURI string) *sarif.Run {
	name := toolName
	if name == "" {
		name = fallback
	}
	drv := sarif.NewDriver(name)
	drv.InformationURI = checks.StringPtr(infoURI)
	return sarif.NewRun(*sarif.NewTool(drv))
}

// resultLocation is a one-file region. End line and column are omitted when <= 0.
func resultLocation(uri string, startLine, startColumn, endLine, endColumn int) *sarif.Location {
	region := sarif.NewRegion().
		WithStartLine(startLine).
		WithStartColumn(startColumn)
	if endLine > 0 {
		region.WithEndLine(endLine)
	}
	if endColumn > 0 {
		region.WithEndColumn(endColumn)
	}
	return sarif.NewLocation().
		WithPhysicalLocation(sarif.NewPhysicalLocation().
			WithArtifactLocation(sarif.NewArtifactLocation().WithUri(uri)).
			WithRegion(region))
}
