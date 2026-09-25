package audio

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/modot/internal/driver"
)

const step = 0.05

func IncreaseVolume(ctx context.Context) error {
	return adjustVolume(ctx, step)
}

func DecreaseVolume(ctx context.Context) error {
	return adjustVolume(ctx, -step)
}

func adjustVolume(ctx context.Context, delta float64) error {
	vol, err := volume.GetVolume(ctx)
	if err != nil {
		return err
	}
	if err := volume.SetVolume(ctx, driver.Clamp01(vol+delta)); err != nil {
		return err
	}
	return ShowStatus(ctx)
}

func ToggleMute(ctx context.Context) error {
	if err := volume.ToggleMute(ctx); err != nil {
		return err
	}
	return ShowStatus(ctx)
}
