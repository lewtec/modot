package screenshot

import (
	"errors"

	lewscreenshot "github.com/lewtec/lewkit/x/driver/screenshot"
)

var (
	ErrSelectionToolNotFound = lewscreenshot.ErrSelectionToolNotFound
	ErrEmptySelection        = lewscreenshot.ErrEmptySelection
	ErrDirNotConfigured      = errors.New("screenshot dir not configured")
	ErrUnknownTargetType     = lewscreenshot.ErrUnknownTargetType
)

type TargetType = lewscreenshot.TargetType

const (
	TargetAll       = lewscreenshot.TargetAll
	TargetOutput    = lewscreenshot.TargetOutput
	TargetWindow    = lewscreenshot.TargetWindow
	TargetSelection = lewscreenshot.TargetSelection
)

type Driver = lewscreenshot.Driver
