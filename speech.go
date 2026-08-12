package transcript

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mpsanders/VoxScripta/internal/ytdlp"
)

// Audio is acquired media supplied to a Transcriber. Data must be non-nil.
// Format is a container/file-extension hint rather than a codec guarantee.
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

// YTDLPAudioSource acquires the best available audio-only stream through yt-dlp.
// It inspects duration before downloading, applies a download-size limit when
// configured, and keeps downloaded data in isolated temporary storage until
// the returned audio stream is closed.
type YTDLPAudioSource struct {
	client *ytdlp.Client
}

// NewYTDLPAudioSource constructs an audio source using executable as the
// yt-dlp path or command name. An empty executable selects "yt-dlp".
func NewYTDLPAudioSource(executable string) *YTDLPAudioSource {
	return &YTDLPAudioSource{client: ytdlp.NewClient(executable, nil)}
}

// Acquire downloads checked audio for videoID or a supported YouTube URL.
// A positive duration limit rejects unknown, live, and known over-limit media
// before download. MaxBytes is passed to yt-dlp as a best-effort transfer guard
// and strictly checked against the final file. Closing returned Data removes
// its temporary artifact and can report a cleanup error.
func (s *YTDLPAudioSource) Acquire(ctx context.Context, videoID string, options AudioOptions) (Audio, error) {
	if ctx == nil {
		return Audio{}, fmt.Errorf("%w: context must not be nil", ErrInvalidInput)
	}
	if s == nil || s.client == nil {
		return Audio{}, fmt.Errorf("%w: yt-dlp audio source is not configured", ErrInvalidInput)
	}
	normalizedVideoID, err := ParseVideoID(videoID)
	if err != nil {
		return Audio{}, err
	}
	if options.MaxDuration < 0 {
		return Audio{}, fmt.Errorf("%w: maximum audio duration must not be negative", ErrInvalidInput)
	}
	if options.MaxBytes < 0 {
		return Audio{}, fmt.Errorf("%w: maximum audio size must not be negative", ErrInvalidInput)
	}
	artifact, err := s.client.AcquireAudio(ctx, normalizedVideoID, options.MaxDuration, options.MaxBytes)
	if err != nil {
		if ctx != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return Audio{}, ctxErr
			}
		}
		if errors.Is(err, ytdlp.ErrAudioLimit) {
			return Audio{}, fmt.Errorf("%w: %v", ErrLimitExceeded, err)
		}
		return Audio{}, classifyYTDLPError("acquire audio", err)
	}
	return Audio{Data: artifact.Data, Format: artifact.Format, Duration: artifact.Duration, Size: artifact.Size}, nil
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

// Get acquires checked audio for videoID, closes it after transcription,
// returns cleanup failures, and produces a normalized speech-to-text
// Transcript. Automatic-caption policy is irrelevant and is ignored.
func (p SpeechToTextProvider) Get(ctx context.Context, videoID string, options Options) (result Transcript, returnedErr error) {
	if ctx == nil {
		return Transcript{}, fmt.Errorf("%w: context must not be nil", ErrInvalidInput)
	}
	if p.AudioSource == nil || p.Transcriber == nil {
		return Transcript{}, fmt.Errorf("%w: audio source and transcriber must not be nil", ErrInvalidInput)
	}
	if p.MaxDuration < 0 || p.MaxBytes < 0 {
		return Transcript{}, fmt.Errorf("%w: audio limits must not be negative", ErrInvalidInput)
	}
	normalizedVideoID, err := ParseVideoID(videoID)
	if err != nil {
		return Transcript{}, err
	}

	audio, err := p.AudioSource.Acquire(ctx, normalizedVideoID, AudioOptions{MaxDuration: p.MaxDuration, MaxBytes: p.MaxBytes})
	if err != nil {
		return Transcript{}, err
	}
	if audio.Data == nil {
		return Transcript{}, fmt.Errorf("%w: audio source returned nil data", ErrProviderFailure)
	}
	defer func() {
		if err := audio.Data.Close(); err != nil {
			result = Transcript{}
			returnedErr = errors.Join(returnedErr, fmt.Errorf("%w: close acquired audio: %v", ErrProviderFailure, err))
		}
	}()
	if audio.Duration < 0 || audio.Size < 0 {
		return Transcript{}, fmt.Errorf("%w: audio source returned invalid metadata", ErrProviderFailure)
	}
	if p.MaxDuration > 0 {
		if audio.Duration == 0 {
			return Transcript{}, fmt.Errorf("%w: audio duration is unknown", ErrLimitExceeded)
		}
		if audio.Duration > p.MaxDuration {
			return Transcript{}, fmt.Errorf("%w: audio duration %s exceeds %s", ErrLimitExceeded, audio.Duration, p.MaxDuration)
		}
	}
	if p.MaxBytes > 0 {
		if audio.Size == 0 {
			return Transcript{}, fmt.Errorf("%w: audio size is unknown", ErrLimitExceeded)
		}
		if audio.Size > p.MaxBytes {
			return Transcript{}, fmt.Errorf("%w: audio size %d exceeds %d bytes", ErrLimitExceeded, audio.Size, p.MaxBytes)
		}
	}

	transcription, err := p.Transcriber.Transcribe(ctx, audio, append([]string(nil), options.Languages...))
	if err != nil {
		return Transcript{}, err
	}
	result = Transcript{VideoID: normalizedVideoID, Language: transcription.Language, Source: SourceSpeechToText, Provider: transcription.Provider, Segments: transcription.Segments}
	if err := result.Validate(); err != nil {
		return Transcript{}, fmt.Errorf("%w: transcriber returned invalid transcription: %v", ErrProviderFailure, err)
	}
	if strings.TrimSpace(result.Provider.Name) == "" {
		return Transcript{}, fmt.Errorf("%w: transcriber provider name must not be empty", ErrProviderFailure)
	}
	return result, nil
}
