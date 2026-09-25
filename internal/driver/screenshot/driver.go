package screenshot

import (
	"errors"

	kitscreenshot "github.com/lewtec/lewkit/x/driver/screenshot"
)

var (
	ErrSelectionToolNotFound = kitscreenshot.ErrSelectionToolNotFound
	ErrEmptySelection        = kitscreenshot.ErrEmptySelection
	ErrDirNotConfigured      = errors.New("screenshot dir not configured")
	ErrUnknownTargetType     = kitscreenshot.ErrUnknownTargetType
)

type TargetType = kitscreenshot.TargetType

const (
	TargetAll       = kitscreenshot.TargetAll
	TargetOutput    = kitscreenshot.TargetOutput
	TargetWindow    = kitscreenshot.TargetWindow
	TargetSelection = kitscreenshot.TargetSelection
)

type Driver = kitscreenshot.Driver
