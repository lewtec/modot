//go:build !linux && !windows

package archive

import (
	"errors"
	"io"
)

var errWIMUnsupported = errors.New("wim: unsupported platform")

// ExtractWIM is not available on this platform.
func ExtractWIM(_ io.Reader, _ string, _ int) error {
	return errWIMUnsupported
}
