package power

import (
	"context"
	"fmt"

	lewpower "github.com/lewtec/lewkit/x/driver/power"
	"github.com/lewtec/modot/internal/api"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/logging"
)

func Wake(ctx context.Context, host string) error {
	cfg, err := configcue.LoadForWorkspace(ctx, "")
	if err != nil {
		return err
	}
	var hosts map[string]struct {
		MAC string `json:"mac"`
	}
	if err := cfg.Decode("hosts", &hosts); err != nil {
		return err
	}

	hostCfg, ok := hosts[host]
	if !ok {
		return fmt.Errorf("%w: %s", api.ErrHostNotFound, host)
	}
	if hostCfg.MAC == "" {
		return fmt.Errorf("%w: host %s has no MAC address", api.ErrConfigNotFound, host)
	}
	if err := lewpower.Wake(ctx, hostCfg.MAC); err != nil {
		return err
	}
	logging.GetLogger(ctx).Info("sent Wake-on-LAN magic packet", "host", host, "mac", hostCfg.MAC)
	return nil
}
