package camera

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/modot/internal/driver"
	cameraapi "github.com/lewtec/modot/internal/driver/camera"
)

type List struct{}

func (List) Description() string { return "List cameras" }

func (*List) Run(ctx context.Context) error {
	drv, err := driver.Get[cameraapi.Driver](ctx)
	if err != nil {
		return err
	}
	cams, err := drv.List(ctx)
	if err != nil {
		return err
	}
	if len(cams) == 0 {
		return ErrNoCamerasFound
	}
	for _, cam := range cams {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", cam.ID(), cam.Name())
	}
	return nil
}
