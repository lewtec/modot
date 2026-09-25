package driver

import (
	"strings"

	lewdriver "github.com/lewtec/lewkit/x/driver"
)

// lewIface maps modot capability names onto the lewkit packages that now own them.
var lewIface = map[string]string{
	"github.com/lewtec/modot/internal/driver/audio.Driver":        "github.com/lewtec/lewkit/x/driver/volume.Driver",
	"github.com/lewtec/modot/internal/driver/battery.Driver":      "github.com/lewtec/lewkit/x/driver/battery.Driver",
	"github.com/lewtec/modot/internal/driver/brightness.Driver":   "github.com/lewtec/lewkit/x/driver/brightness.Driver",
	"github.com/lewtec/modot/internal/driver/camera.Driver":       "github.com/lewtec/lewkit/x/driver/camera.Driver",
	"github.com/lewtec/modot/internal/driver/clipboard.Driver":    "github.com/lewtec/lewkit/x/driver/clipboard.Driver",
	"github.com/lewtec/modot/internal/driver/dialog.Chooser":      "github.com/lewtec/lewkit/x/driver/launcher.Chooser",
	"github.com/lewtec/modot/internal/driver/dialog.Confirmer":    "github.com/lewtec/lewkit/x/driver/launcher.Confirmer",
	"github.com/lewtec/modot/internal/driver/dialog.Driver":       "github.com/lewtec/lewkit/x/driver/launcher.Driver",
	"github.com/lewtec/modot/internal/driver/dialog.Prompter":     "github.com/lewtec/lewkit/x/driver/launcher.Prompter",
	"github.com/lewtec/modot/internal/driver/media.Driver":        "github.com/lewtec/lewkit/x/driver/media.Driver",
	"github.com/lewtec/modot/internal/driver/notification.Driver": "github.com/lewtec/lewkit/x/driver/notification.Driver",
	"github.com/lewtec/modot/internal/driver/opener.Driver":       "github.com/lewtec/lewkit/x/driver/opener.Driver",
	"github.com/lewtec/modot/internal/driver/power.Driver":        "github.com/lewtec/lewkit/x/driver/power.Driver",
	"github.com/lewtec/modot/internal/driver/screen.Driver":       "github.com/lewtec/lewkit/x/driver/screen.Driver",
	"github.com/lewtec/modot/internal/driver/screenshot.Driver":   "github.com/lewtec/lewkit/x/driver/screenshot.Driver",
	"github.com/lewtec/modot/internal/driver/terminal.Driver":     "github.com/lewtec/lewkit/x/driver/terminal.Driver",
	"github.com/lewtec/modot/internal/driver/wallpaper.Driver":    "github.com/lewtec/lewkit/x/driver/wallpaper.Driver",
	"github.com/lewtec/modot/internal/driver/wm.Driver":           "github.com/lewtec/lewkit/x/driver/wm.Driver",
}

// lewID maps driver slugs that were renamed when the implementation moved.
var lewID = map[string]string{
	"audio_pulse":         "volume_pulse",
	"brightness_ctl":      "brightness_brightnessctl",
	"screen_wayland_sway": "wm_sway",
	"screen_x11_i3":       "wm_i3",
	"v4l-ffmpeg":          "camera_v4l",
	"wayland_swaybg":      "wallpaper_swaybg",
	"x11_feh":             "wallpaper_feh",
}

// ApplyLewWeights copies CUE driver weights onto lewkit's registry.
// Slugs absent from the CUE map keep the factory Weight.
func ApplyLewWeights(cue map[string]map[string]int) error {
	overlay := map[string]map[string]int{}
	for iface, weights := range cue {
		name := iface
		if mapped, ok := lewIface[iface]; ok {
			name = mapped
		}
		if !strings.HasPrefix(name, "github.com/lewtec/lewkit/x/driver/") {
			continue
		}
		row := overlay[name]
		if row == nil {
			row = map[string]int{}
			overlay[name] = row
		}
		for id, weight := range weights {
			if renamed, ok := lewID[id]; ok {
				id = renamed
			}
			row[id] = weight
		}
	}

	full := map[string]map[string]int{}
	for typ, factories := range lewdriver.Drivers {
		if typ.PkgPath() == "" || typ.Name() == "" {
			continue
		}
		iface := typ.PkgPath() + "." + typ.Name()
		row := map[string]int{}
		for id, factory := range factories {
			if weight, ok := overlay[iface][id]; ok {
				row[id] = weight
				continue
			}
			if weighter, ok := factory.(interface{ Weight() int }); ok {
				row[id] = weighter.Weight()
				continue
			}
			row[id] = 0
		}
		full[iface] = row
	}
	if len(full) == 0 {
		return nil
	}
	return lewdriver.SetWeights(full)
}
