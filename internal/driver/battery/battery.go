package battery

import (
	"context"

	kitbattery "github.com/lewtec/lewkit/x/driver/battery"
)

var ErrNoBattery = kitbattery.ErrNoBattery

type Status = kitbattery.Status

const (
	Charging    = kitbattery.Charging
	Discharging = kitbattery.Discharging
	Full        = kitbattery.Full
	Unknown     = kitbattery.Unknown
)

type Driver = kitbattery.Driver

func BatteryStatus(ctx context.Context) (Status, error) {
	return kitbattery.BatteryStatus(ctx)
}
