package init

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyEmbeddedModulesAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	// Seed a truncated prior file under the example module path if present after copy.
	require.NoError(t, copyEmbeddedModules(dir), "copyEmbeddedModules")
	// Ensure no .tmp leftovers
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".tmp") {
			assert.Fail(t, "temp file left behind: "+path)
		}
		return nil
	})
	require.NoError(t, err)
	// Second copy overwrites cleanly
	require.NoError(t, copyEmbeddedModules(dir), "second copyEmbeddedModules")
	// At least one regular file was written
	var files int
	if walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files++
		}
		return nil
	}); walkErr != nil {
		t.Logf("count walk: %v", walkErr)
	}
	require.Greater(t, files, 0, "expected embedded module files")
}
