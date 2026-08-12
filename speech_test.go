package transcript

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type recordingAudioSource struct {
	audio   Audio
	err     error
	options AudioOptions
}

// Acquire records resource limits and returns the configured audio.
func (s *recordingAudioSource) Acquire(_ context.Context, _ string, options AudioOptions) (Audio, error) {
	s.options = options
	return s.audio, s.err
}

type recordingTranscriber struct {
	result Transcription
	err    error
	calls  int
	hints  []string
}

// Transcribe records language hints and returns the configured result.
func (t *recordingTranscriber) Transcribe(_ context.Context, _ Audio, hints []string) (Transcription, error) {
	t.calls++
	t.hints = hints
	return t.result, t.err
}

type closeRecorder struct {
	io.Reader
	closed bool
}

// Close records that ownership cleanup occurred.
func (r *closeRecorder) Close() error {
	r.closed = true
	return nil
}

// TestSpeechToTextProviderGet verifies composition, limits, validation, and
// cleanup behavior for speech-to-text fallback acquisition.
func TestSpeechToTextProviderGet(t *testing.T) {
	valid := Transcription{Language: Language{Code: "en"}, Provider: ProviderMetadata{Name: "test-stt"}, Segments: []Segment{{Start: 0, End: time.Second, Text: "hello"}}}
	tests := []struct {
		name        string
		ctx         context.Context
		source      AudioSource
		transcriber Transcriber
		maxDuration time.Duration
		maxBytes    int64
		wantErr     error
		wantCalls   int
		wantClosed  bool
	}{
		{name: "successful transcription", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}, Format: "wav", Duration: time.Minute, Size: 5}}, transcriber: &recordingTranscriber{result: valid}, maxDuration: time.Hour, maxBytes: 10, wantCalls: 1, wantClosed: true},
		{name: "zero limits allow unknown metadata", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("")}}}, transcriber: &recordingTranscriber{result: valid}, wantCalls: 1, wantClosed: true},
		{name: "duration exceeds limit", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}, Duration: 2 * time.Minute}}, transcriber: &recordingTranscriber{result: valid}, maxDuration: time.Minute, wantErr: ErrLimitExceeded, wantClosed: true},
		{name: "size exceeds limit", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}, Size: 11}}, transcriber: &recordingTranscriber{result: valid}, maxBytes: 10, wantErr: ErrLimitExceeded, wantClosed: true},
		{name: "nil audio data", ctx: context.Background(), source: &recordingAudioSource{}, transcriber: &recordingTranscriber{result: valid}, wantErr: ErrProviderFailure},
		{name: "audio source failure", ctx: context.Background(), source: &recordingAudioSource{err: ErrProviderFailure}, transcriber: &recordingTranscriber{result: valid}, wantErr: ErrProviderFailure},
		{name: "transcriber failure", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}}}, transcriber: &recordingTranscriber{err: context.Canceled}, wantErr: context.Canceled, wantCalls: 1, wantClosed: true},
		{name: "invalid transcription", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}}}, transcriber: &recordingTranscriber{result: Transcription{}}, wantErr: ErrProviderFailure, wantCalls: 1, wantClosed: true},
		{name: "nil context", source: &recordingAudioSource{}, transcriber: &recordingTranscriber{result: valid}, wantErr: ErrInvalidInput},
		{name: "nil source", ctx: context.Background(), transcriber: &recordingTranscriber{result: valid}, wantErr: ErrInvalidInput},
		{name: "nil transcriber", ctx: context.Background(), source: &recordingAudioSource{}, wantErr: ErrInvalidInput},
		{name: "negative limits", ctx: context.Background(), source: &recordingAudioSource{}, transcriber: &recordingTranscriber{result: valid}, maxBytes: -1, wantErr: ErrInvalidInput},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := SpeechToTextProvider{AudioSource: test.source, Transcriber: test.transcriber, MaxDuration: test.maxDuration, MaxBytes: test.maxBytes}
			got, err := provider.Get(test.ctx, "abcdefghijk", Options{Languages: []string{"en-AU", "en"}})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Get() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			transcriber, hasRecorder := test.transcriber.(*recordingTranscriber)
			if hasRecorder && transcriber.calls != test.wantCalls {
				t.Fatalf("transcriber calls = %d, want %d", transcriber.calls, test.wantCalls)
			}
			if test.wantErr == nil && (got.Source != SourceSpeechToText || got.VideoID != "abcdefghijk" || !hasRecorder || len(transcriber.hints) != 2) {
				t.Fatalf("Get() = %#v, transcriber = %#v", got, transcriber)
			}
			if source, ok := test.source.(*recordingAudioSource); ok {
				if closer, ok := source.audio.Data.(*closeRecorder); ok && closer.closed != test.wantClosed {
					t.Fatalf("audio closed = %v, want %v", closer.closed, test.wantClosed)
				}
			}
		})
	}
}
