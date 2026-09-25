package driver_test

import (
	"context"
	"testing"

	lewdriver "github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/lewkit/x/driver/terminal"
	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/modot/internal/driver"
	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"

	_ "github.com/lewtec/modot/internal/driver/audio/prelude"
	_ "github.com/lewtec/modot/internal/driver/dialog/prelude"
	_ "github.com/lewtec/modot/internal/driver/notification/prelude"
	_ "github.com/lewtec/modot/internal/driver/terminal/prelude"
)

func TestApplyKitWeightsTranslatesLegacyKeys(t *testing.T) {
	err := driver.ApplyKitWeights(map[string]map[string]int{
		"github.com/lewtec/modot/internal/driver/audio.Driver": {
			"audio_pulse": 101,
		},
	})
	require.Error(t, err)

	ctx := logging.NewWriterContext(t.Output())
	err = driver.ApplyKitWeights(map[string]map[string]int{
		"github.com/lewtec/modot/internal/driver/audio.Driver": {
			"audio_pulse": 80,
		},
		"github.com/lewtec/modot/internal/driver/dialog.Chooser": {
			"rofi":     100,
			"terminal": 50,
		},
		"github.com/lewtec/modot/internal/driver/notification.Driver": {
			"notification_dbus":        100,
			"notification_notify_send": 10,
		},
		"github.com/lewtec/modot/internal/driver/terminal.Driver": {
			"terminal_kitty": 70,
		},
	})
	require.NoError(t, err)

	assertWeight[volume.Driver](t, ctx, "volume_pulse", 80)
	assertWeight[launcher.Chooser](t, ctx, "rofi", 100)
	assertWeight[launcher.Chooser](t, ctx, "terminal", 50)
	assertWeight[notification.Driver](t, ctx, "notification_dbus", 100)
	assertWeight[notification.Driver](t, ctx, "notification_notify_send", 10)
	assertWeight[terminal.Driver](t, ctx, "terminal_kitty", 70)
}

func assertWeight[T any](t *testing.T, ctx context.Context, id string, want int) {
	t.Helper()
	handles, err := lewdriver.List[T](ctx)
	if err != nil {
		t.Logf("no compatible %s driver: %v", id, err)
		return
	}
	for _, handle := range handles {
		if handle.ID == id {
			require.Equal(t, want, handle.Weight)
			return
		}
	}
	t.Logf("driver %s is not compatible in this environment", id)
}
