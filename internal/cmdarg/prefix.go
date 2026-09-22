package cmdarg

import (
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	lewpath "github.com/lewtec/lewkit/x/path"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
)

// HomePrefix is home apply and home plan --prefix.
// The default is ~, the user home directory.
type HomePrefix struct {
	value string
}

func (HomePrefix) ArgDefault() string { return "~" }

func (p HomePrefix) Value() string { return p.value }

func (p *HomePrefix) Parse(arg string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("%w: %w", cmd.ErrInvalidArgument, err)
	}
	if arg == "" || arg == "~" {
		p.value = home
		return nil
	}
	return assignAbsolute(&p.value, envdriver.ExpandPathIn(arg, home))
}

// SystemPrefix is system apply --prefix.
// The default is /.
type SystemPrefix struct {
	value string
}

func (SystemPrefix) ArgDefault() string { return "/" }

func (p SystemPrefix) Value() string { return p.value }

func (p *SystemPrefix) Parse(arg string) error {
	if arg == "" {
		arg = "/"
	}
	return assignAbsolute(&p.value, arg)
}

func assignAbsolute(dest *string, arg string) error {
	name := lewpath.New(arg)
	if !name.IsAbs() {
		return fmt.Errorf("%w: prefix must be absolute", cmd.ErrInvalidArgument)
	}
	*dest = name.String()
	return nil
}

var (
	_ cmd.Parser       = (*HomePrefix)(nil)
	_ cmd.Parser       = (*SystemPrefix)(nil)
	_ cmd.Arg[string]  = (*HomePrefix)(nil)
	_ cmd.Arg[string]  = (*SystemPrefix)(nil)
	_ cmd.ArgDefaulter = HomePrefix{}
	_ cmd.ArgDefaulter = SystemPrefix{}
)
