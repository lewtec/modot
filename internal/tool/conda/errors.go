package conda

import "errors"

var (
	// ErrEmptyRef is returned when a conda ref is blank.
	ErrEmptyRef = errors.New("conda ref cannot be empty")
	// ErrInvalidRef is returned when a conda ref or match spec cannot be parsed.
	ErrInvalidRef = errors.New("invalid conda ref")
	// ErrInvalidVersion is returned when a version string is not a conda version.
	ErrInvalidVersion = errors.New("invalid conda version")
	// ErrPackageNotFound is returned when the channel index has no such package.
	ErrPackageNotFound = errors.New("conda package not found")
	// ErrNoBuild is returned when the package has no build for this platform.
	ErrNoBuild = errors.New("no conda build for this platform")
	// ErrUnsatisfied is returned when a run dependency cannot be selected.
	ErrUnsatisfied = errors.New("conda dependency not satisfied")
	// ErrUnknownArchive is returned when a download is neither .conda nor .tar.bz2.
	ErrUnknownArchive = errors.New("unknown conda archive")
	// ErrPathEscapes is returned when an archive member resolves outside the prefix.
	ErrPathEscapes = errors.New("conda archive path escapes destination")
	// ErrPrefixTooLong is returned when a binary prefix placeholder is shorter than the install prefix.
	ErrPrefixTooLong = errors.New("install prefix is longer than the conda placeholder")
)
