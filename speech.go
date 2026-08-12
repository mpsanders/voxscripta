package transcript

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// Audio is acquired media supplied to a Transcriber. Data must be non-nil.
// Duration and Size describe the complete media when known; zero means unknown
// or empty. The SpeechToTextProvider closes Data after transcription returns.
type Audio struct {
	Data     io.ReadCloser
	Format   string
	Duration time.Duration
	Size     int64
}

// AudioOptions describes constraints an AudioSource should apply while
// acquiring media. A zero limit means no limit. Sources should enforce these
// limits during acquisition where possible; SpeechToTextProvider checks the
// returned metadata again before transcription.
type AudioOptions struct {
	MaxDuration time.Duration
	MaxBytes    int64
}

// AudioSource acquires audio for videoID. Implementations must honor ctx and
// return an Audio whose Data is owned by the caller. Implementations shared
// between goroutines must be safe for concurrent use.
type AudioSource interface {
	Acquire(ctx context.Context, videoID string, options AudioOptions) (Audio, error)
}

// Transcription is provider output before it is attached to video metadata.
// Segments must satisfy the same ordering and validation rules as Transcript.
type Transcription struct {
	Language Language
	Provider ProviderMetadata
	Segments []Segment
}

// Transcriber converts acquired audio into timestamped text. languageHints
// are ordered caller preferences and may be empty. It must not close audio.
// Implementations shared between goroutines must be safe for concurrent use.
type Transcriber interface {
	Transcribe(ctx context.Context, audio Audio, languageHints []string) (Transcription, error)
}

// SpeechToTextProvider composes separate audio acquisition and transcription
// implementations into a Provider suitable for explicit fallback use.
type SpeechToTextProvider struct {
	AudioSource AudioSource
	Transcriber Transcriber
	MaxDuration time.Duration
	MaxBytes    int64
}

// Get acquires bounded audio for videoID, guarantees its closure, transcribes
// it, and returns a normalized speech-to-text Transcript. Automatic-caption
// policy is irrelevant to speech transcription and is ignored.
func (p SpeechToTextProvider) Get(ctx context.Context, videoID string, options Options) (Transcript, error) {
	if ctx == nil {
		return Transcript{}, fmt.Errorf("%w: context must not be nil", ErrInvalidInput)
	}
	if p.AudioSource == nil || p.Transcriber == nil {
		return Transcript{}, fmt.Errorf("%w: audio source and transcriber must not be nil", ErrInvalidInput)
	}
	if p.MaxDuration < 0 || p.MaxBytes < 0 {
		return Transcript{}, fmt.Errorf("%w: audio limits must not be negative", ErrInvalidInput)
	}

	audio, err := p.AudioSource.Acquire(ctx, videoID, AudioOptions{MaxDuration: p.MaxDuration, MaxBytes: p.MaxBytes})
	if err != nil {
		return Transcript{}, err
	}
	if audio.Data == nil {
		return Transcript{}, fmt.Errorf("%w: audio source returned nil data", ErrProviderFailure)
	}
	defer audio.Data.Close()
	if audio.Duration < 0 || audio.Size < 0 {
		return Transcript{}, fmt.Errorf("%w: audio source returned invalid metadata", ErrProviderFailure)
	}
	if p.MaxDuration > 0 && audio.Duration > p.MaxDuration {
		return Transcript{}, fmt.Errorf("%w: audio duration %s exceeds %s", ErrLimitExceeded, audio.Duration, p.MaxDuration)
	}
	if p.MaxBytes > 0 && audio.Size > p.MaxBytes {
		return Transcript{}, fmt.Errorf("%w: audio size %d exceeds %d bytes", ErrLimitExceeded, audio.Size, p.MaxBytes)
	}

	result, err := p.Transcriber.Transcribe(ctx, audio, append([]string(nil), options.Languages...))
	if err != nil {
		return Transcript{}, err
	}
	transcript := Transcript{VideoID: videoID, Language: result.Language, Source: SourceSpeechToText, Provider: result.Provider, Segments: result.Segments}
	if err := transcript.Validate(); err != nil {
		return Transcript{}, fmt.Errorf("%w: transcriber returned invalid transcription: %v", ErrProviderFailure, err)
	}
	if strings.TrimSpace(transcript.Provider.Name) == "" {
		return Transcript{}, fmt.Errorf("%w: transcriber provider name must not be empty", ErrProviderFailure)
	}
	return transcript, nil
}
