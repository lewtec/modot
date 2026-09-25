package power

import (
	"context"
	"fmt"
	kitpower "github.com/lewtec/lewkit/x/driver/power"
	"github.com/lewtec/modot/internal/api"
	"github.com/lewtec/modot/internal/configcue"
	"github.com/lewtec/modot/internal/logging"
	"net"
)

func Lock(ctx context.Context) error {
	return kitpower.Lock(ctx)
}

func Reboot(ctx context.Context) error {
	return kitpower.Reboot(ctx)
}

func Shutdown(ctx context.Context) error {
	return kitpower.Shutdown(ctx)
}

func Suspend(ctx context.Context) error {
	return kitpower.Suspend(ctx)
}

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
	macStr := ""
	if !ok {
		return fmt.Errorf("%w: %s", api.ErrHostNotFound, host)
	} else {
		macStr = hostCfg.MAC
	}

	if macStr == "" {
		return fmt.Errorf("%w: host %s has no MAC address", api.ErrConfigNotFound, host)
	}

	hwAddr, err := net.ParseMAC(macStr)
	if err != nil {
		return fmt.Errorf("%w: %s (%w)", api.ErrInvalidAddr, macStr, err)
	}

	packet := make([]byte, 6+16*6)
	for i := range 6 {
		packet[i] = 0xFF
	}
	for i := 1; i <= 16; i++ {
		copy(packet[i*6:(i+1)*6], hwAddr)
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", "255.255.255.255:9")
	if err != nil {
		return fmt.Errorf("dial UDP broadcast: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logging.ReportError(ctx, err)
		}
	}()

	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("send magic packet: %w", err)
	}

	logger := logging.GetLogger(ctx)
	logger.Info("sent Wake-on-LAN magic packet", "host", host, "mac", macStr)
	return nil
}
