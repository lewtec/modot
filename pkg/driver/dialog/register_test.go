package dialog_test

import (
	"testing"

	"github.com/lucasew/workspaced/pkg/driver"
	_ "github.com/lucasew/workspaced/pkg/driver/dialog/rofi"
	_ "github.com/lucasew/workspaced/pkg/driver/dialog/wofi"
	"github.com/stretchr/testify/require"
)

func TestRofiAndWofiRegisterBothInterfaces(t *testing.T) {
	shape := driver.RegisteredWeightShape()
	want := []string{"rofi", "wofi"}
	for _, iface := range []string{
		"github.com/lucasew/workspaced/pkg/driver/dialog.Chooser",
		"github.com/lucasew/workspaced/pkg/driver/dialog.Driver",
	} {
		require.ElementsMatch(t, want, shape[iface])
	}
}
