package core

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/modot/internal/module"
)

func TestBase16IconsLinuxRegistered(t *testing.T) {
	t.Parallel()
	_, ok := module.GetCoreModule("base16-icons-linux")
	require.True(t, ok, `GetCoreModule("base16-icons-linux") missing; source file must not be named *_linux.go (implicit GOOS build tag)`)
}
