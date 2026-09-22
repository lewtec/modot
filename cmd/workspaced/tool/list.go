package tool

import (
	"context"
	"fmt"

	lewtool "github.com/lewtec/lewkit/x/tool"
	"github.com/lucasew/workspaced/internal/tool"
)

type List struct{}

func (List) Description() string { return "List installed tools" }

func (*List) Run(ctx context.Context) error {
	dir, err := tool.GetToolsDir()
	if err != nil {
		return err
	}
	store, err := lewtool.Open(dir)
	if err != nil {
		return err
	}
	tools, err := store.ListInstalled()
	if err != nil {
		return err
	}
	for _, t := range tools {
		fmt.Printf("%s %s\n", t.Name, t.Version)
	}
	return nil
}
