package wallpaper

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/modot/internal/driver/wallpaper"
)

type Command struct {
	Apod   *Apod
	Change *Change
	Video  *Video
}

func (Command) Description() string {
	return "Wallpaper management"
}

type Apod struct{}

func (Apod) Description() string {
	return "Fetch NASA Astronomy Picture of the Day and set as wallpaper"
}

func (*Apod) Run(ctx context.Context) error {
	return wallpaper.SetAPOD(ctx)
}

type Change struct {
	path *cmd.StringArg
}

func (Change) Description() string {
	return "Change wallpaper to a random image or specific path"
}

func (c *Change) Run(ctx context.Context) error {
	path := ""
	if c.path != nil {
		path = c.path.Value()
	}
	return wallpaper.SetStatic(ctx, path)
}

type Video struct {
	path cmd.StringArg
}

func (Video) Description() string { return "Set an animated video as wallpaper" }

func (c *Video) Run(ctx context.Context) error {
	return wallpaper.SetAnimated(ctx, c.path.Value())
}
