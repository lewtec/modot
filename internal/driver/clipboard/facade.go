package clipboard

import (
	"context"
	"image"

	kitclip "github.com/lewtec/lewkit/x/driver/clipboard"
)

func WriteImage(ctx context.Context, img image.Image) error {
	return kitclip.WriteImage(ctx, img)
}

func WriteText(ctx context.Context, text string) error {
	return kitclip.WriteText(ctx, text)
}
