package codebase

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/stretchr/testify/require"
)

func TestWriteSarifAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lint.sarif")
	require.NoError(t, os.WriteFile(path, []byte("{broken"), 0o644))
	report := &sarif.Report{Version: "2.1.0"}
	require.NoError(t, writeSarifAtomic(path, report))
	_, err := os.Stat(path + ".tmp")
	require.ErrorIs(t, err, fs.ErrNotExist, "tmp left")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.Equal(t, byte('{'), raw[0])
}
