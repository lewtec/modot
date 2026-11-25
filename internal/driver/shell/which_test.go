package shell_test

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/modot/internal/driver"
	_ "github.com/lewtec/modot/internal/driver/exec/native"
	"github.com/lewtec/modot/internal/driver/shell"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestWhichDriverExposesPathOnly(t *testing.T) {
	shell.RegisterWhich("shell_test_which", "Test sh", "sh")

	ctx := logging.NewWriterContext(t.Output())
	// Force the test registration regardless of weights.
	t.Setenv("MODOT_FORCE_SHELL_DRIVER", "shell_test_which")

	d, err := shell.Get(ctx)
	require.NoError(t, err)
	path, err := d.Path(ctx)
	require.NoError(t, err)
	require.Equal(t, "sh", filepath.Base(path))

	_, ok := d.(driver.DriverFactory[shell.Driver])
	require.False(t, ok, "shell.Driver unexpectedly implements DriverFactory")
}
