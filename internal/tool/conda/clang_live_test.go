package conda

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/stretchr/testify/require"
)

func TestLiveClangLinksZlib(t *testing.T) {
	if os.Getenv("MODOT_TEST_CONDA") == "" {
		t.Skip("set MODOT_TEST_CONDA=1 to build a zlib program with conda-forge clang")
	}

	zlibDir := shortPrefix(t, "cz-zlib")
	clangDir := shortPrefix(t, "cz-clang")

	zlib, err := (&Backend{}).Tool("conda-forge/zlib")
	require.NoError(t, err)
	require.NoError(t, zlib.Install(t.Context(), "latest", zlibDir))
	require.FileExists(t, filepath.Join(zlibDir, "include", "zlib.h"))

	clang, err := (&Backend{}).Tool("conda-forge/clang")
	require.NoError(t, err)
	require.NoError(t, clang.Install(t.Context(), "latest", clangDir))
	clangBin := filepath.Join(clangDir, "bin", "clang")
	require.FileExists(t, clangBin)

	src := filepath.Join(t.TempDir(), "app.c")
	program := "#include <stdio.h>\n#include <zlib.h>\nint main(void){printf(\"zlib %s\\n\", zlibVersion());return 0;}\n"
	require.NoError(t, os.WriteFile(src, []byte(program), 0o644))

	out := filepath.Join(t.TempDir(), "app")
	cmd, err := execdriver.Command(
		clangBin,
		"-I"+filepath.Join(zlibDir, "include"),
		"-L"+filepath.Join(zlibDir, "lib"),
		"-Wl,-rpath,"+filepath.Join(zlibDir, "lib"),
		"-o", out,
		src,
		"-lz",
	)
	require.NoError(t, err)
	compileOut, err := combined(t.Context(), cmd)
	require.NoError(t, err, "compile failed:\n%s", compileOut)

	run, err := execdriver.Command(out)
	require.NoError(t, err)
	got, err := combined(t.Context(), run)
	require.NoError(t, err, "run failed:\n%s", got)
	require.Contains(t, string(got), "zlib ")
}

func combined(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := execdriver.Run(ctx, cmd)
	return buf.Bytes(), err
}

func shortPrefix(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join("/tmp", name)
	require.NoError(t, os.RemoveAll(dir))
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
