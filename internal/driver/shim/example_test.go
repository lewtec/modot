package shim_test

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/modot/internal/driver/prelude"
	"github.com/lewtec/modot/internal/driver/shim"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShimGeneration(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())

	// Create temp directory for test
	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "test-shim")

	// Generate a shim that runs "echo hello world"
	err := shim.Generate(ctx, shimPath, []string{"echo", "hello", "world"})
	require.NoError(t, err)

	// Verify file exists and is executable
	info, err := os.Stat(shimPath)
	require.NoError(t, err)

	assert.NotZero(t, info.Mode()&0111, "Shim file is not executable: %o", info.Mode())

	// Read shim content
	content, err := os.ReadFile(shimPath)
	require.NoError(t, err)

	t.Logf("Generated shim:\n%s", string(content))

	// Verify it's a bash script
	assert.True(t, len(content) >= 2 && content[0] == '#' && content[1] == '!', "Shim doesn't have a shebang")
}

func TestShimWithSpecialCharacters(t *testing.T) {
	ctx := logging.NewWriterContext(t.Output())

	tmpDir := t.TempDir()
	shimPath := filepath.Join(tmpDir, "special-shim")

	// Test with arguments containing spaces and special chars
	err := shim.Generate(ctx, shimPath, []string{"echo", "hello world", "$VAR", "it's test"})
	require.NoError(t, err)

	content, err := os.ReadFile(shimPath)
	require.NoError(t, err)

	t.Logf("Generated shim with special chars:\n%s", string(content))
}
