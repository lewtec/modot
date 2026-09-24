package clipboard_test

import (
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	_ "github.com/lucasew/workspaced/pkg/driver/clipboard/termux"
	_ "github.com/lucasew/workspaced/pkg/driver/clipboard/wlcopy"
	_ "github.com/lucasew/workspaced/pkg/driver/clipboard/xclip"
	"github.com/stretchr/testify/require"
)

func TestBuiltinClipboardIDs(t *testing.T) {
	shape := driver.RegisteredWeightShape()
	ids := shape["github.com/lucasew/workspaced/pkg/driver/clipboard.Driver"]
	require.ElementsMatch(t, []string{
		"clipboard_termux",
		"clipboard_wlcopy",
		"clipboard_xclip",
	}, ids)
}
