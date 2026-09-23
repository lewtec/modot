package shell_test

import (
	"path/filepath"
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	_ "github.com/lucasew/workspaced/pkg/driver/exec/native"
	"github.com/lucasew/workspaced/pkg/driver/shell"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestWhichDriverExposesPathOnly(t *testing.T) {
	shell.RegisterWhich("shell_test_which", "Test sh", "sh")

	ctx := logging.NewWriterContext(t.Output())
	// Force the test registration regardless of weights.
	t.Setenv("WORKSPACED_FORCE_SHELL_DRIVER", "shell_test_which")

	d, err := shell.Get(ctx)
	require.NoError(t, err)
	path, err := d.Path(ctx)
	require.NoError(t, err)
	require.Equal(t, "sh", filepath.Base(path))

	_, ok := d.(driver.DriverFactory[shell.Driver])
	require.False(t, ok, "shell.Driver unexpectedly implements DriverFactory")
}
