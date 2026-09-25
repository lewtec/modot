package termux

import (
	"context"
	"fmt"
	"strings"

	kitdriver "github.com/lewtec/lewkit/x/driver"
	kitterminal "github.com/lewtec/lewkit/x/driver/terminal"
	"github.com/lewtec/modot/internal/driver"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

func init() {
	kitdriver.Register[kitterminal.Driver](factory{})
}

type factory struct{}

func (factory) ID() string   { return "terminal_termux" }
func (factory) Name() string { return "Termux" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.RequireTermux()
}

func (factory) New(context.Context) (kitterminal.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Open(ctx context.Context, opts kitterminal.Options) error {
	if opts.Command == "" {
		return execdriver.MustRun(ctx, "am", "start", "--user", "0", "-n", "com.termux/.app.TermuxActivity").Run()
	}

	fullCmd := opts.Command
	if !strings.HasPrefix(fullCmd, "/") {
		if path, err := execdriver.Which(ctx, fullCmd); err == nil {
			fullCmd = path
		}
	}

	if len(opts.Args) > 0 {
		var escapedArgs []string
		for _, arg := range opts.Args {
			escapedArgs = append(escapedArgs, fmt.Sprintf("%q", arg))
		}
		fullCmd += " " + strings.Join(escapedArgs, " ")
	}

	return execdriver.MustRun(ctx, "am", "startservice",
		"--user", "0",
		"-n", "com.termux/com.termux.app.TermuxService",
		"-a", "com.termux.service_execute",
		"-e", "com.termux.execute.command", fullCmd,
	).Run()
}
