package dialog

import (
	"context"

	"github.com/lucasew/workspaced/pkg/driver"
)

// pairFactory registers one concrete driver as both Chooser and Driver.
type pairFactory[T any] struct {
	id, name string
	compat   func(context.Context) error
	new      func() T
}

func (f *pairFactory[T]) ID() string   { return f.id }
func (f *pairFactory[T]) Name() string { return f.name }

func (f *pairFactory[T]) CheckCompatibility(ctx context.Context) error {
	if f.compat == nil {
		return nil
	}
	return f.compat(ctx)
}

func (f *pairFactory[T]) New(context.Context) (T, error) {
	return f.new(), nil
}

// RegisterChooserAndDriver registers the same id, name, and compatibility
// check for dialog.Chooser and dialog.Driver. newDriver builds the impl.
func RegisterChooserAndDriver(id, name string, compat func(context.Context) error, newDriver func() Driver) {
	driver.Register[Chooser](&pairFactory[Chooser]{
		id:     id,
		name:   name,
		compat: compat,
		new:    func() Chooser { return newDriver() },
	})
	driver.Register[Driver](&pairFactory[Driver]{
		id:     id,
		name:   name,
		compat: compat,
		new:    newDriver,
	})
}
