// Package transcript provides provider-independent types and operations for
// acquiring normalized, timestamped YouTube transcripts.
package transcript

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SourceKind identifies how transcript text was produced.
type SourceKind string

const (
	// SourceManual identifies captions supplied by a video's creator.
	SourceManual SourceKind = "manual"
	// SourceAutomatic identifies captions generated automatically by YouTube.
	SourceAutomatic SourceKind = "automatic"
	// SourceSpeechToText identifies captions generated from the video's audio.
	SourceSpeechToText SourceKind = "speech_to_text"
)

// Segment is one normalized, half-open interval of transcript text. Overlap
// with adjacent segments is permitted because caption formats commonly use it.
type Segment struct {
	Start time.Duration `json:"start"`
	End   time.Duration `json:"end"`
	Text  string        `json:"text"`
}

// Validate checks that the segment has non-negative, increasing timestamps
// and contains non-whitespace text.
func (s Segment) Validate() error {
	if s.Start < 0 {
		return fmt.Errorf("%w: segment start must not be negative", ErrInvalidInput)
	}
	if s.End < s.Start {
		return fmt.Errorf("%w: segment end precedes start", ErrInvalidInput)
	}
	if strings.TrimSpace(s.Text) == "" {
		return fmt.Errorf("%w: segment text must not be empty", ErrInvalidInput)
	}
	return nil
}

// Language describes the selected caption language. Code should be a BCP 47
// language tag when the provider supplies one; Name is optional display text.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

// ProviderMetadata contains safe diagnostic details about acquisition. Values
// must not contain signed URLs, cookies, credentials, or other secrets.
type ProviderMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Transcript is a normalized transcript whose segments are ordered by Start.
type Transcript struct {
	VideoID  string           `json:"video_id"`
	Title    string           `json:"title,omitempty"`
	Language Language         `json:"language"`
	Source   SourceKind       `json:"source"`
	Provider ProviderMetadata `json:"provider"`
	Segments []Segment        `json:"segments"`
}

// Validate verifies the transcript's required metadata and segment invariants.
// Overlapping segments are accepted, but segment start times must not decrease.
func (t Transcript) Validate() error {
	if _, err := ParseVideoID(t.VideoID); err != nil {
		return fmt.Errorf("%w: transcript video ID: %v", ErrInvalidInput, err)
	}
	if strings.TrimSpace(t.Language.Code) == "" {
		return fmt.Errorf("%w: language code must not be empty", ErrInvalidInput)
	}
	if !t.Source.valid() {
		return fmt.Errorf("%w: invalid source kind %q", ErrInvalidInput, t.Source)
	}
	if len(t.Segments) == 0 {
		return fmt.Errorf("%w: transcript must contain at least one segment", ErrInvalidInput)
	}
	for i, segment := range t.Segments {
		if err := segment.Validate(); err != nil {
			return fmt.Errorf("segment %d: %w", i, err)
		}
		if i > 0 && segment.Start < t.Segments[i-1].Start {
			return fmt.Errorf("%w: segment %d starts before segment %d", ErrInvalidInput, i, i-1)
		}
	}
	return nil
}

// Text renders the normalized segments as newline-separated plain text.
func (t Transcript) Text() string {
	lines := make([]string, 0, len(t.Segments))
	for _, segment := range t.Segments {
		text := strings.TrimSpace(segment.Text)
		if text != "" {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

// valid reports whether s is one of the defined transcript source kinds.
func (s SourceKind) valid() bool {
	switch s {
	case SourceManual, SourceAutomatic, SourceSpeechToText:
		return true
	default:
		return false
	}
}

// Options describes caller preferences for transcript acquisition. Languages
// are evaluated in order; an empty list asks the provider for the original or
// otherwise best available language.
type Options struct {
	Languages      []string
	AllowAutomatic bool
}

// Provider acquires and normalizes a transcript for videoID. Implementations
// must honor cancellation through ctx and must be safe for concurrent use when
// documented as such.
type Provider interface {
	Get(ctx context.Context, videoID string, options Options) (Transcript, error)
}
