package cmdarg

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
)

type prefixKey struct{}

// WithPrefix stores root for callers that are not a command.
// A command field tagged ctx:"prefix" is the usual path.
func WithPrefix(ctx context.Context, root *lewpath.Root) context.Context {
	return context.WithValue(ctx, prefixKey{}, root)
}

// Prefix is the apply root opened by --prefix.
func Prefix(ctx context.Context) *lewpath.Root {
	if ctx == nil {
		return nil
	}
	if root, ok := cmd.Lookup[*lewpath.Root](ctx, "prefix"); ok && root != nil {
		return root
	}
	root, _ := ctx.Value(prefixKey{}).(*lewpath.Root)
	return root
}

// PrefixPath is Prefix.Name, or empty when no root is on ctx.
func PrefixPath(ctx context.Context) string {
	root := Prefix(ctx)
	if root == nil {
		return ""
	}
	return root.Name()
}

// HomePrefix is home apply and home plan --prefix.
// The default is ~, the user home directory.
// Value is that directory opened as a rooted filesystem.
type HomePrefix struct {
	root *lewpath.Root
}

func (HomePrefix) ArgDefault() string { return "~" }

func (p HomePrefix) Value() *lewpath.Root { return p.root }

func (p *HomePrefix) Parse(arg string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("%w: %w", cmd.ErrInvalidArgument, err)
	}
	if arg == "" || arg == "~" {
		arg = home
	} else {
		arg = envdriver.ExpandPathIn(arg, home)
	}
	return p.set(arg)
}

// SystemPrefix is system apply --prefix.
// The default is /.
// Value is that directory opened as a rooted filesystem.
type SystemPrefix struct {
	root *lewpath.Root
}

func (SystemPrefix) ArgDefault() string { return "/" }

func (p SystemPrefix) Value() *lewpath.Root { return p.root }

func (p *SystemPrefix) Parse(arg string) error {
	if arg == "" {
		arg = "/"
	}
	return p.set(arg)
}

func (p *HomePrefix) set(arg string) error {
	root, err := openPrefix(arg)
	if err != nil {
		return err
	}
	if p.root != nil {
		_ = p.root.Close()
	}
	p.root = root
	return nil
}

func (p *SystemPrefix) set(arg string) error {
	root, err := openPrefix(arg)
	if err != nil {
		return err
	}
	if p.root != nil {
		_ = p.root.Close()
	}
	p.root = root
	return nil
}

func openPrefix(arg string) (*lewpath.Root, error) {
	name := lewpath.New(arg)
	if !name.IsAbs() {
		return nil, fmt.Errorf("%w: prefix must be absolute", cmd.ErrInvalidArgument)
	}
	root, err := lewpath.Open(name.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", cmd.ErrInvalidArgument, err)
	}
	return root, nil
}

var (
	_ cmd.Parser             = (*HomePrefix)(nil)
	_ cmd.Parser             = (*SystemPrefix)(nil)
	_ cmd.Arg[*lewpath.Root] = (*HomePrefix)(nil)
	_ cmd.Arg[*lewpath.Root] = (*SystemPrefix)(nil)
	_ cmd.ArgDefaulter       = HomePrefix{}
	_ cmd.ArgDefaulter       = SystemPrefix{}
)
