package core

import "github.com/lewtec/modot/internal/placestep"

// place step / path errors (tabled for %w + errors.Is).
var (
	errPlaceRequireNoMatch = placestep.ErrRequireNoMatch
	errPlaceRequireMustNot = placestep.ErrRequireMustNot
	errPlaceBadPattern     = placestep.ErrBadPattern
	errPlaceEmptyPath      = placestep.ErrEmptyPath
	errPlacePathEscape     = placestep.ErrPathEscape
	errPlaceMoveFrom       = placestep.ErrMoveFrom
	errPlaceMoveTo         = placestep.ErrMoveTo
)
