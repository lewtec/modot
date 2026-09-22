package tool

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	lewtool "github.com/lewtec/lewkit/x/tool"
)

type Versions struct {
	spec cmd.StringArg
}

func (Versions) Description() string {
	return "List available versions for a tool ref from upstream"
}

func (v *Versions) Run(ctx context.Context) error {
	versions, err := listVersions(ctx, v.spec.Value())
	if err != nil {
		return err
	}
	for _, ver := range versions {
		fmt.Fprintln(os.Stdout, ver)
	}
	return nil
}

func listVersions(ctx context.Context, specStr string) ([]string, error) {
	spec, err := lewtool.Parse(specStr)
	if err != nil {
		return nil, err
	}
	backend, err := lewtool.Get(spec.Backend)
	if err != nil {
		return nil, err
	}
	installed, err := backend.Tool(spec.Package)
	if err != nil {
		return nil, err
	}
	return installed.ListVersions(ctx)
}
