package terminal_test

import (
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	_ "github.com/lucasew/workspaced/pkg/driver/terminal/alacritty"
	_ "github.com/lucasew/workspaced/pkg/driver/terminal/foot"
	_ "github.com/lucasew/workspaced/pkg/driver/terminal/kitty"
	_ "github.com/lucasew/workspaced/pkg/driver/terminal/termux"
	"github.com/stretchr/testify/require"
)

func TestBuiltinTerminalIDs(t *testing.T) {
	shape := driver.RegisteredWeightShape()
	ids := shape["github.com/lucasew/workspaced/pkg/driver/terminal.Driver"]
	require.ElementsMatch(t, []string{
		"terminal_alacritty",
		"terminal_foot",
		"terminal_kitty",
		"terminal_termux",
	}, ids)
}
