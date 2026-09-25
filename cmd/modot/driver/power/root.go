package power

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	kitpower "github.com/lewtec/lewkit/x/driver/power"
	"github.com/lewtec/modot/internal/driver/power"
)

type Command struct {
	Lock     *Lock
	Reboot   *Reboot
	Shutdown *Shutdown
	Suspend  *Suspend
	Wake     *Wake
}

func (Command) Description() string {
	return "Power management commands"
}

type Lock struct{}

func (Lock) Description() string { return "Lock the session" }
func (*Lock) Run(ctx context.Context) error {
	return kitpower.Lock(ctx)
}

type Reboot struct{}

func (Reboot) Description() string { return "Reboot the system" }
func (*Reboot) Run(ctx context.Context) error {
	return kitpower.Reboot(ctx)
}

type Shutdown struct{}

func (Shutdown) Description() string { return "Power off the system" }
func (*Shutdown) Run(ctx context.Context) error {
	return kitpower.Shutdown(ctx)
}

type Suspend struct{}

func (Suspend) Description() string { return "Suspend the system" }
func (*Suspend) Run(ctx context.Context) error {
	return kitpower.Suspend(ctx)
}

type Wake struct {
	host cmd.StringArg
}

func (Wake) Description() string { return "Send Wake-on-LAN magic packet" }
func (w *Wake) Run(ctx context.Context) error {
	return power.Wake(ctx, w.host.Value())
}
