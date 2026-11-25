package configcue

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/modot/internal/logging"
)

func TestLegacyFileIsWarnedAndNotLoaded(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", root)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	require.NoError(t, os.WriteFile(filepath.Join(root, "modot.cue"), []byte("package picuinha\n\ndesktop: dark_mode: true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "workspaced.cue"), []byte("package old\n\ndesktop: dark_mode: false\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "workspaced.lock.json"), []byte("{}\n"), 0o644))

	var logs bytes.Buffer
	ctx := logging.NewWriterContext(&logs)
	cfg, err := LoadForWorkspace(ctx, root)
	require.NoError(t, err)

	desktop, _ := cfg.Raw()["desktop"].(map[string]any)
	require.Equal(t, true, desktop["dark_mode"])
	require.Contains(t, logs.String(), "workspaced.cue")
	require.Contains(t, logs.String(), "workspaced.lock.json")
	require.NotContains(t, logs.String(), "false")
}

func TestLegacyEnvIsWarnedAndNotRead(t *testing.T) {
	t.Setenv("WORKSPACED_NO_CACHE", "secret-value")
	var logs bytes.Buffer
	WarnLegacyEnv(logging.NewWriterContext(&logs))
	text := logs.String()
	require.Contains(t, text, "WORKSPACED_NO_CACHE")
	require.Contains(t, text, "MODOT_NO_CACHE")
	require.NotContains(t, text, "secret-value")
}
