package filespine

import (
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/stretchr/testify/require"
)

func TestOpenDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	opened, err := OpenDir(lewpath.New(dir))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, opened.Close()) })
	require.Equal(t, dir, opened.Name())

	_, err = OpenDir(lewpath.New("relative"))
	require.EqualError(t, err, "directory is not absolute")
}
