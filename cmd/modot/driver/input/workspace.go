package input

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/lewtec/lewkit/x/cmd"
	lewdriver "github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
	lewwm "github.com/lewtec/lewkit/x/driver/wm"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/filespine"
)

type Workspace struct {
	Move cmd.Flag `long:"move" help:"Move container to workspace"`
}

func (Workspace) Description() string { return "Workspace switcher" }

func (c *Workspace) Run(ctx context.Context) error {
	result, err := configcue.Evaluate(ctx, configcue.DiscoverOptions{
		HomeLayers: true,
		Mode:       filespine.ModeHome,
	})
	if err != nil {
		return err
	}
	var raw struct {
		Workspaces map[string]int `json:"workspaces"`
	}
	if err := json.Unmarshal(result.JSON, &raw); err != nil {
		return fmt.Errorf("decode evaluated config: %w", err)
	}

	var items []launcher.Item
	var keys []string
	for k := range raw.Workspaces {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		items = append(items, launcher.Item{
			Label: k,
			Value: strconv.Itoa(raw.Workspaces[k]),
		})
	}

	d, err := lewdriver.Get[launcher.Driver](ctx)
	if err != nil {
		return err
	}

	selected, err := d.Choose(ctx, launcher.ChooseOptions{
		Prompt: "Workspace",
		Items:  items,
	})
	if err != nil {
		return err
	}

	if selected == nil {
		return nil
	}

	return lewwm.SwitchToWorkspace(ctx, selected.Value, c.Move.Value())
}
