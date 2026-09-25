package brightness

import (
	"context"

	kitbrightness "github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/lewtec/modot/internal/driver"
)

const step = 0.05

func IncreaseBrightness(ctx context.Context) error {
	return adjustBrightness(ctx, step)
}

func DecreaseBrightness(ctx context.Context) error {
	return adjustBrightness(ctx, -step)
}

func adjustBrightness(ctx context.Context, delta float64) error {
	status, err := kitbrightness.Status(ctx)
	if err != nil {
		return err
	}
	if err := kitbrightness.SetBrightness(ctx, driver.Clamp01(status.Brightness+delta)); err != nil {
		return err
	}
	return ShowStatus(ctx)
}
