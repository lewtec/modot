package battery

import (
	"context"

	lewbattery "github.com/lewtec/lewkit/x/driver/battery"
)

var ErrNoBattery = lewbattery.ErrNoBattery

type Status = lewbattery.Status

const (
	Charging    = lewbattery.Charging
	Discharging = lewbattery.Discharging
	Full        = lewbattery.Full
	Unknown     = lewbattery.Unknown
)

type Driver = lewbattery.Driver

func BatteryStatus(ctx context.Context) (Status, error) {
	return lewbattery.BatteryStatus(ctx)
}
