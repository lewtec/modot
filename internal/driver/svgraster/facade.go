package svgraster

import (
	"context"
	"image"

	lewdriver "github.com/lewtec/lewkit/x/driver"
)

func Ensure(ctx context.Context) error {
	return lewdriver.With(ctx, func(d Driver) error { return d.Ensure(ctx) })
}

func RasterizeSVG(ctx context.Context, svg string, width int, height int) (image.Image, error) {
	return lewdriver.WithResult(ctx, func(d Driver) (image.Image, error) {
		return d.RasterizeSVG(ctx, svg, width, height)
	})
}
