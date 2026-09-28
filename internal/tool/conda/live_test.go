package conda

import (
	"os"
	"path/filepath"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	_ "github.com/lewtec/lewkit/x/driver/fetchurl/native"
	_ "github.com/lewtec/lewkit/x/driver/httpclient/native"
	"github.com/stretchr/testify/require"
)

func TestLiveRipgrepFromCondaForge(t *testing.T) {
	if os.Getenv("MODOT_TEST_CONDA") == "" {
		t.Skip("set MODOT_TEST_CONDA=1 to install ripgrep from conda-forge")
	}
	tool, err := (&Backend{}).Tool("conda-forge/ripgrep")
	require.NoError(t, err)
	versions, err := tool.ListVersions(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	dest := t.TempDir()
	require.NoError(t, tool.Install(t.Context(), versions[0], dest))
	bin := filepath.Join(dest, "bin", "rg")
	_, err = os.Stat(bin)
	require.NoError(t, err)

	cmd, err := execdriver.Command(bin, "--version")
	require.NoError(t, err)
	out, err := execdriver.Output(t.Context(), cmd)
	require.NoError(t, err)
	require.Contains(t, string(out), "ripgrep")
}
