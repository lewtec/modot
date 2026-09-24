package terminal

import (
	"context"

	"github.com/lucasew/workspaced/pkg/driver"
	execdriver "github.com/lucasew/workspaced/pkg/driver/exec"
)

// execFactory starts one terminal binary. Alacritty, foot, and kitty share
// this; only id, name, flags, and an optional env check differ.
type execFactory struct {
	id, name, binary, titleFlag string
	commandAsE                  bool
	compat                      func(context.Context) error
}

func (f *execFactory) ID() string   { return f.id }
func (f *execFactory) Name() string { return f.name }

func (f *execFactory) CheckCompatibility(ctx context.Context) error {
	if f.compat != nil {
		if err := f.compat(ctx); err != nil {
			return err
		}
	}
	return execdriver.RequireBinary(ctx, f.binary)
}

func (f *execFactory) New(context.Context) (Driver, error) {
	return &execDriver{
		binary:     f.binary,
		titleFlag:  f.titleFlag,
		commandAsE: f.commandAsE,
	}, nil
}

type execDriver struct {
	binary, titleFlag string
	commandAsE        bool
}

func (d *execDriver) Open(ctx context.Context, opts Options) error {
	cmd := execdriver.MustRun(ctx, d.binary, BuildOpenArgs(opts, d.titleFlag, d.commandAsE)...)
	return cmd.Start()
}

// RegisterExec registers a terminal that starts binary with BuildOpenArgs.
// titleFlag is the argv flag for Options.Title (for example "-T").
// commandAsE inserts "-e" before Options.Command when true.
// compat runs before the binary-on-PATH check; nil means PATH only.
func RegisterExec(id, name, binary, titleFlag string, commandAsE bool, compat func(context.Context) error) {
	driver.Register[Driver](&execFactory{
		id:         id,
		name:       name,
		binary:     binary,
		titleFlag:  titleFlag,
		commandAsE: commandAsE,
		compat:     compat,
	})
}
