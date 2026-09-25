package termux

import (
	"context"
	"fmt"
	"image"
	"os"
	"strings"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	lewclip "github.com/lewtec/lewkit/x/driver/clipboard"
	dapi "github.com/lewtec/modot/internal/api"
	"github.com/lewtec/modot/internal/driver"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

func init() {
	lewdriver.Register[lewclip.Driver](factory{})
}

type factory struct{}

func (factory) ID() string   { return "clipboard_termux" }
func (factory) Name() string { return "Termux" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if os.Getenv("TERMUX_VERSION") == "" && !execdriver.IsBinaryAvailable(ctx, "termux-clipboard-set") {
		return fmt.Errorf("%w: termux not detected", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (lewclip.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) WriteImage(context.Context, image.Image) error {
	return fmt.Errorf("%w: writing images to clipboard is not supported on Termux", dapi.ErrNotSupported)
}

func (backend) WriteText(ctx context.Context, text string) error {
	if !execdriver.IsBinaryAvailable(ctx, "termux-clipboard-set") {
		return fmt.Errorf("%w: termux-clipboard-set (install termux-api)", dapi.ErrBinaryNotFound)
	}
	cmd := execdriver.MustRun(ctx, "termux-clipboard-set")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
