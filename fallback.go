package transcript

import (
	"context"
	"errors"
	"fmt"
)

// FallbackProvider tries Primary first and invokes Fallback only when Primary
// reports ErrTranscriptUnavailable. This makes potentially expensive or
// privacy-sensitive fallback acquisition explicit and prevents infrastructure,
// cancellation, dependency, and invalid-input failures from triggering it.
// Both providers must be safe for concurrent use if the FallbackProvider is
// shared between goroutines.
type FallbackProvider struct {
	Primary  Provider
	Fallback Provider
}

// Get asks the primary provider to acquire videoID with options. If and only
// if that provider returns ErrTranscriptUnavailable, Get asks the fallback
// provider with the same context, video ID, and options.
func (p FallbackProvider) Get(ctx context.Context, videoID string, options Options) (Transcript, error) {
	if p.Primary == nil {
		return Transcript{}, fmt.Errorf("%w: fallback provider primary must not be nil", ErrInvalidInput)
	}
	if p.Fallback == nil {
		return Transcript{}, fmt.Errorf("%w: fallback provider fallback must not be nil", ErrInvalidInput)
	}

	result, err := p.Primary.Get(ctx, videoID, options)
	if err == nil || !errors.Is(err, ErrTranscriptUnavailable) {
		return result, err
	}
	return p.Fallback.Get(ctx, videoID, options)
}
