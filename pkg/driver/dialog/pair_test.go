package dialog

import (
	"context"
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

type fakeDriver struct{}

func (fakeDriver) Choose(context.Context, ChooseOptions) (*Item, error) {
	return &Item{Value: "ok"}, nil
}
func (fakeDriver) RunApp(context.Context) error       { return nil }
func (fakeDriver) SwitchWindow(context.Context) error { return nil }

func TestPairFactoryBuildsBothShapes(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())
	checks := 0
	compat := func(context.Context) error {
		checks++
		return driver.ErrIncompatible
	}
	chooser := &pairFactory[Chooser]{
		id:     "dialog_test_chooser",
		name:   "Test",
		compat: compat,
		new:    func() Chooser { return fakeDriver{} },
	}
	full := &pairFactory[Driver]{
		id:     "dialog_test_driver",
		name:   "Test",
		compat: compat,
		new:    func() Driver { return fakeDriver{} },
	}

	require.Equal(t, "dialog_test_chooser", chooser.ID())
	require.Equal(t, "Test", full.Name())
	require.ErrorIs(t, chooser.CheckCompatibility(ctx), driver.ErrIncompatible)
	require.ErrorIs(t, full.CheckCompatibility(ctx), driver.ErrIncompatible)
	require.Equal(t, 2, checks)

	gotChooser, err := chooser.New(ctx)
	require.NoError(t, err)
	item, err := gotChooser.Choose(ctx, ChooseOptions{})
	require.NoError(t, err)
	require.Equal(t, "ok", item.Value)

	gotDriver, err := full.New(ctx)
	require.NoError(t, err)
	require.NoError(t, gotDriver.RunApp(ctx))
	require.NoError(t, gotDriver.SwitchWindow(ctx))
}
