package transcript

import "errors"

var (
	// ErrInvalidInput indicates that caller-supplied input is invalid.
	ErrInvalidInput = errors.New("invalid input")
	// ErrMissingDependency indicates that a required external executable is unavailable.
	ErrMissingDependency = errors.New("missing dependency")
	// ErrTranscriptUnavailable indicates that no acceptable transcript could be acquired.
	ErrTranscriptUnavailable = errors.New("transcript unavailable")
	// ErrUnsupportedFormat indicates that a caption or output format is unsupported.
	ErrUnsupportedFormat = errors.New("unsupported format")
	// ErrProviderFailure indicates that an acquisition provider failed unexpectedly.
	ErrProviderFailure = errors.New("provider failure")
	// ErrLimitExceeded indicates that configured resource limits prevented acquisition.
	ErrLimitExceeded = errors.New("limit exceeded")
)
