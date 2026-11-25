package power

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
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
	return power.Lock(ctx)
}

type Reboot struct{}

func (Reboot) Description() string { return "Reboot the system" }
func (*Reboot) Run(ctx context.Context) error {
	return power.Reboot(ctx)
}

type Shutdown struct{}

func (Shutdown) Description() string { return "Power off the system" }
func (*Shutdown) Run(ctx context.Context) error {
	return power.Shutdown(ctx)
}

type Suspend struct{}

func (Suspend) Description() string { return "Suspend the system" }
func (*Suspend) Run(ctx context.Context) error {
	return power.Suspend(ctx)
}

type Wake struct {
	host cmd.StringArg
}

func (Wake) Description() string { return "Send Wake-on-LAN magic packet" }
func (w *Wake) Run(ctx context.Context) error {
	return power.Wake(ctx, w.host.Value())
}
