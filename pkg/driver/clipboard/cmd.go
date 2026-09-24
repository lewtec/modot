package clipboard

import (
	"context"
	"image"

	"github.com/lucasew/workspaced/pkg/driver"
	execdriver "github.com/lucasew/workspaced/pkg/driver/exec"
)

// cmdFactory writes clipboard payloads by spawning one binary.
// xclip and wl-copy share this; only id, name, args, and an optional env check differ.
type cmdFactory struct {
	id, name, binary string
	compat           func(context.Context) error
	imageArgs        []string
	textArgs         []string
}

func (f *cmdFactory) ID() string   { return f.id }
func (f *cmdFactory) Name() string { return f.name }

func (f *cmdFactory) CheckCompatibility(ctx context.Context) error {
	if f.compat != nil {
		if err := f.compat(ctx); err != nil {
			return err
		}
	}
	return execdriver.RequireBinary(ctx, f.binary)
}

func (f *cmdFactory) New(context.Context) (Driver, error) {
	return &cmdDriver{
		binary:    f.binary,
		imageArgs: append([]string(nil), f.imageArgs...),
		textArgs:  append([]string(nil), f.textArgs...),
	}, nil
}

type cmdDriver struct {
	binary    string
	imageArgs []string
	textArgs  []string
}

func (d *cmdDriver) WriteImage(ctx context.Context, img image.Image) error {
	return WriteImageViaCmd(ctx, img, d.binary, d.imageArgs...)
}

func (d *cmdDriver) WriteText(ctx context.Context, text string) error {
	return WriteTextViaCmd(ctx, text, d.binary, d.textArgs...)
}

// RegisterCmd registers a clipboard driver that writes PNG and text to binary.
// imageArgs and textArgs are the flags after the binary name.
// compat runs before the binary-on-PATH check; nil means PATH only.
func RegisterCmd(id, name, binary string, compat func(context.Context) error, imageArgs, textArgs []string) {
	driver.Register[Driver](&cmdFactory{
		id:        id,
		name:      name,
		binary:    binary,
		compat:    compat,
		imageArgs: append([]string(nil), imageArgs...),
		textArgs:  append([]string(nil), textArgs...),
	})
}
