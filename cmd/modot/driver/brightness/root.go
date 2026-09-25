package brightness

import (
	"context"

	"github.com/lewtec/modot/internal/driver/brightness"
)

type Command struct {
	Up     *Up
	Down   *Down
	Show   *Show
	Status *Show `cmd:"status"`
}

func (Command) Description() string {
	return "Control screen brightness"
}

type Up struct{}

func (Up) Description() string { return "Increase brightness" }
func (*Up) Run(ctx context.Context) error {
	return brightness.IncreaseBrightness(ctx)
}

type Down struct{}

func (Down) Description() string { return "Decrease brightness" }
func (*Down) Run(ctx context.Context) error {
	return brightness.DecreaseBrightness(ctx)
}

type Show struct{}

func (Show) Description() string { return "Show current brightness" }
func (*Show) Run(ctx context.Context) error {
	return brightness.ShowStatus(ctx)
}
