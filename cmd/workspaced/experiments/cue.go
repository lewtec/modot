package experiments

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/workspaced/internal/configcue"
)

type Cue struct {
	Layers *CueLayers `cmd:"layers"`
	Export *CueExport `cmd:"export"`
}

func (Cue) Description() string {
	return "Inspect experimental layered CUE configuration"
}

type CueLayers struct {
	cwd cmd.WorkDirArg `help:"Start directory for workspaced.cue discovery"`
}

func (CueLayers) Description() string { return "List discovered workspaced.cue layers" }

func (c *CueLayers) Run(ctx context.Context) error {
	res, err := configcue.DiscoverLayers(ctx, configcue.DiscoverOptions{Cwd: c.cwd.Value()})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

type CueExport struct {
	cwd cmd.WorkDirArg `help:"Start directory for workspaced.cue discovery"`
}

func (CueExport) Description() string { return "Export the unified experimental CUE config as JSON" }

func (c *CueExport) Run(ctx context.Context) error {
	data, err := configcue.ExportJSON(ctx, configcue.DiscoverOptions{Cwd: c.cwd.Value()})
	if err != nil {
		return err
	}
	var pretty any
	if err := json.Unmarshal(data, &pretty); err != nil {
		return fmt.Errorf("decode generated json: %w", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(pretty)
}
