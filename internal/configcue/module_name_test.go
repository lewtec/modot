package configcue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateModuleName(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"base16", "base16_gtk", "theme", "_private", "a1"} {
		require.NoError(t, ValidateModuleName(name), "name %q", name)
	}
	for _, name := range []string{"base16-gtk", "foo.bar", "1theme", "", "with space"} {
		require.Error(t, ValidateModuleName(name), "name %q", name)
	}
}
