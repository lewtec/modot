package driver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

type probe interface {
	ID() string
}

type probeFactory struct{}

func (probeFactory) ID() string                               { return "probe_test" }
func (probeFactory) Name() string                             { return "Probe" }
func (probeFactory) CheckCompatibility(context.Context) error { return nil }
func (probeFactory) New(context.Context) (probe, error)       { return probeImpl{}, nil }

type probeImpl struct{}

func (probeImpl) ID() string { return "probe_test" }

func TestWithAndWithResult(t *testing.T) {
	driver.Register[probe](probeFactory{})
	t.Setenv("WORKSPACED_FORCE_DRIVER_TEST_PROBE_DRIVER", "probe_test")
	ctx := logging.NewWriterContext(t.Output())

	err := driver.With(ctx, func(p probe) error {
		if p.ID() != "probe_test" {
			return errors.New("bad id")
		}
		return nil
	})
	require.NoError(t, err)

	id, err := driver.WithResult(ctx, func(p probe) (string, error) { return p.ID(), nil })
	require.NoError(t, err)
	require.Equal(t, "probe_test", id)
}
