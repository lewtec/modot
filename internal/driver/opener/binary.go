package opener

import (
	"context"

	kitdriver "github.com/lewtec/lewkit/x/driver"
	kitopener "github.com/lewtec/lewkit/x/driver/opener"
	execdriver "github.com/lewtec/modot/internal/driver/exec"
)

type binaryFactory struct {
	id, name, binary string
	compat           func(context.Context) error
}

func (f *binaryFactory) ID() string   { return f.id }
func (f *binaryFactory) Name() string { return f.name }
func (f *binaryFactory) Weight() int  { return 60 }

func (f *binaryFactory) CheckCompatibility(ctx context.Context) error {
	if f.compat != nil {
		if err := f.compat(ctx); err != nil {
			return err
		}
	}
	return execdriver.RequireBinary(ctx, f.binary)
}

func (f *binaryFactory) New(context.Context) (kitopener.Driver, error) {
	return &binaryDriver{binary: f.binary}, nil
}

type binaryDriver struct{ binary string }

func (d *binaryDriver) Open(ctx context.Context, target string) error {
	return execdriver.MustRun(ctx, d.binary, target).Start()
}

// RegisterBinary registers an opener that runs binary with the target path/URL.
// compat runs before the binary-on-PATH check; nil means PATH check only.
func RegisterBinary(id, name, binary string, compat func(context.Context) error) {
	kitdriver.Register[kitopener.Driver](&binaryFactory{id: id, name: name, binary: binary, compat: compat})
}
