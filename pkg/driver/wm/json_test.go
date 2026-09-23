package wm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	dapi "github.com/lucasew/workspaced/pkg/api"
	"github.com/lucasew/workspaced/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestJSONViaCmd(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())

	path := filepath.Join(t.TempDir(), "out.json")
	err := os.WriteFile(path, []byte(`{"name":"HDMI-A-1","x":12}`), 0o644)
	require.NoError(t, err)

	got, err := JSONViaCmd[struct {
		Name string `json:"name"`
		X    int    `json:"x"`
	}](ctx, "cat", path)
	require.NoError(t, err)
	require.Equal(t, "HDMI-A-1", got.Name)
	require.Equal(t, 12, got.X)
}

func TestJSONViaCmdFailed(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	_, err := JSONViaCmd[struct{}](ctx, "false")
	require.Error(t, err, "expected command failure")
	require.ErrorIs(t, err, dapi.ErrIPC)
	var ee *exec.ExitError
	require.ErrorAs(t, err, &ee)
}

func TestJSONViaCmdBadJSON(t *testing.T) {
	t.Parallel()
	ctx := logging.NewWriterContext(t.Output())
	path := filepath.Join(t.TempDir(), "not.json")
	err := os.WriteFile(path, []byte("not-json"), 0o644)
	require.NoError(t, err)
	_, err = JSONViaCmd[struct{}](ctx, "cat", path)
	require.Error(t, err, "expected decode failure")
	require.ErrorIs(t, err, dapi.ErrIPC)
	var se *json.SyntaxError
	require.ErrorAs(t, err, &se)
}
