package cmdarg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	envdriver "github.com/lewtec/modot/internal/driver/env"
)

type prefixKey struct{}

// Prefix is a cmd.DataDirArg. A leading ~ expands to the user home directory
// before the standard directory check. The default lives on the command
// field: ~ for home, . for codebase, / for system.
type Prefix struct {
	cmd.DataDirArg
}

func (p *Prefix) Parse(arg string) error {
	expanded, err := expandHome(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", cmd.ErrInvalidArgument, err)
	}
	return p.DataDirArg.Parse(expanded)
}

func expandHome(arg string) (string, error) {
	if arg != "~" && !strings.HasPrefix(arg, "~/") {
		return arg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if arg == "~" {
		return home, nil
	}
	return envdriver.ExpandPathIn(arg, home), nil
}

// WithPrefix stores a directory for callers that are not a command.
// A command field tagged ctx:"prefix" is the usual path.
func WithPrefix(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, prefixKey{}, dir)
}

// PrefixPath is the --prefix directory from the command context.
func PrefixPath(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if dir, ok := cmd.Lookup[string](ctx, "prefix"); ok && dir != "" {
		return dir
	}
	dir, ok := ctx.Value(prefixKey{}).(string)
	if !ok {
		return ""
	}
	return dir
}
