package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProgressCountsLines(t *testing.T) {
	w := newBuildProgress(nil)
	w.total = 3
	_, err := w.Write([]byte("github.com/a\ngithub.com/b\n"))
	require.NoError(t, err)
	_, err = w.Write([]byte("github.com/c\n"))
	require.NoError(t, err)
	require.Equal(t, int64(3), w.n)
	require.Equal(t, int64(3), w.total)
	require.Equal(t, "github.com/a\ngithub.com/b\ngithub.com/c", w.tail())
}

func TestFindBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Helper to create file
	createFile := func(path string, mode os.FileMode) {
		dir := filepath.Dir(path)
		require.NoError(t, os.MkdirAll(dir, 0755))
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, os.Chmod(path, mode))
	}

	t.Run("Exact match modot", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "exact")
		require.NoError(t, os.MkdirAll(dir, 0755))
		createFile(filepath.Join(dir, "modot"), 0755)

		found, err := findBinary(dir)
		require.NoError(t, err)
		assert.Equal(t, "modot", filepath.Base(found))
	})

	t.Run("Exact match modot.exe", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "exact_exe")
		require.NoError(t, os.MkdirAll(dir, 0755))
		createFile(filepath.Join(dir, "modot.exe"), 0755)

		found, err := findBinary(dir)
		require.NoError(t, err)
		assert.Equal(t, "modot.exe", filepath.Base(found))
	})

	t.Run("Match in bin/", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "in_bin")
		require.NoError(t, os.MkdirAll(dir, 0755))
		createFile(filepath.Join(dir, "bin", "modot"), 0755)

		found, err := findBinary(dir)
		require.NoError(t, err)
		assert.Equal(t, "modot", filepath.Base(found))
	})

	t.Run("Fallback scan single binary", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "fallback_single")
		require.NoError(t, os.MkdirAll(dir, 0755))

		binName := "modot-custom"
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		createFile(filepath.Join(dir, binName), 0755)
		createFile(filepath.Join(dir, "README.md"), 0644)

		found, err := findBinary(dir)
		require.NoError(t, err)
		assert.Equal(t, binName, filepath.Base(found))
	})

	// Platform specific fallback tests
	if runtime.GOOS != "windows" {
		t.Run("Fallback scan executable bit", func(t *testing.T) {
			dir := filepath.Join(tmpDir, "fallback_exec")
			require.NoError(t, os.MkdirAll(dir, 0755))

			createFile(filepath.Join(dir, "not_exec"), 0644)
			createFile(filepath.Join(dir, "is_exec"), 0755) // +x

			found, err := findBinary(dir)
			require.NoError(t, err)
			assert.Equal(t, "is_exec", filepath.Base(found))
		})
	} else {
		t.Run("Fallback scan exe extension", func(t *testing.T) {
			dir := filepath.Join(tmpDir, "fallback_exe_ext")
			require.NoError(t, os.MkdirAll(dir, 0755))

			createFile(filepath.Join(dir, "not_exe"), 0755)
			createFile(filepath.Join(dir, "is_exe.exe"), 0755)

			found, err := findBinary(dir)
			require.NoError(t, err)
			assert.Equal(t, "is_exe.exe", filepath.Base(found))
		})
	}

	t.Run("No binary found", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "none")
		require.NoError(t, os.MkdirAll(dir, 0755))
		createFile(filepath.Join(dir, "README.md"), 0644)

		_, err := findBinary(dir)
		require.Error(t, err)
	})
}
