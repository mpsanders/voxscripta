package transcript

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mpsanders/VoxScripta/internal/ytdlp"
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
	err    error
}

// Close records that ownership cleanup occurred.
func (r *closeRecorder) Close() error {
	r.closed = true
	return r.err
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
		{name: "unknown duration with limit", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}, Size: 5}}, transcriber: &recordingTranscriber{result: valid}, maxDuration: time.Minute, wantErr: ErrLimitExceeded, wantClosed: true},
		{name: "unknown size with limit", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}, Duration: time.Second}}, transcriber: &recordingTranscriber{result: valid}, maxBytes: 10, wantErr: ErrLimitExceeded, wantClosed: true},
		{name: "nil audio data", ctx: context.Background(), source: &recordingAudioSource{}, transcriber: &recordingTranscriber{result: valid}, wantErr: ErrProviderFailure},
		{name: "audio source failure", ctx: context.Background(), source: &recordingAudioSource{err: ErrProviderFailure}, transcriber: &recordingTranscriber{result: valid}, wantErr: ErrProviderFailure},
		{name: "transcriber failure", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}}}, transcriber: &recordingTranscriber{err: context.Canceled}, wantErr: context.Canceled, wantCalls: 1, wantClosed: true},
		{name: "invalid transcription", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio")}}}, transcriber: &recordingTranscriber{result: Transcription{}}, wantErr: ErrProviderFailure, wantCalls: 1, wantClosed: true},
		{name: "audio close failure", ctx: context.Background(), source: &recordingAudioSource{audio: Audio{Data: &closeRecorder{Reader: strings.NewReader("audio"), err: errors.New("close failed")}}}, transcriber: &recordingTranscriber{result: valid}, wantErr: ErrProviderFailure, wantCalls: 1, wantClosed: true},
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

type publicAudioRunner struct {
	metadata  string
	contents  string
	err       error
	args      [][]string
	directory string
}

// Run emulates metadata inspection and one isolated audio download.
func (r *publicAudioRunner) Run(ctx context.Context, _ string, args ...string) (ytdlp.CommandResult, error) {
	r.args = append(r.args, append([]string(nil), args...))
	if err := ctx.Err(); err != nil {
		return ytdlp.CommandResult{}, err
	}
	if r.err != nil {
		return ytdlp.CommandResult{}, r.err
	}
	for _, argument := range args {
		if argument == "--dump-single-json" {
			return ytdlp.CommandResult{Stdout: []byte(r.metadata)}, nil
		}
	}
	for index, argument := range args {
		if argument == "--output" && index+1 < len(args) {
			path := strings.Replace(args[index+1], "%(ext)s", "webm", 1)
			r.directory = filepath.Dir(path)
			if err := os.WriteFile(path, []byte(r.contents), 0o600); err != nil {
				return ytdlp.CommandResult{}, err
			}
			return ytdlp.CommandResult{}, nil
		}
	}
	return ytdlp.CommandResult{}, errors.New("unexpected command")
}

// TestYTDLPAudioSourceAcquire verifies the exported adapter's validation,
// normalization, public error mapping, and successful ownership contract.
func TestYTDLPAudioSourceAcquire(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name        string
		source      *YTDLPAudioSource
		ctx         context.Context
		input       string
		options     AudioOptions
		runnerErr   error
		metadata    string
		contents    string
		wantErr     error
		wantVideoID string
	}{
		{name: "normalizes YouTube URL", ctx: context.Background(), input: "https://youtu.be/abcdefghijk", metadata: `{"id":"abcdefghijk","duration":2}`, contents: "audio", wantVideoID: "abcdefghijk"},
		{name: "duration limit mapping", ctx: context.Background(), input: "abcdefghijk", options: AudioOptions{MaxDuration: time.Second}, metadata: `{"id":"abcdefghijk","duration":2}`, wantErr: ErrLimitExceeded},
		{name: "nil receiver", ctx: context.Background(), input: "abcdefghijk", wantErr: ErrInvalidInput},
		{name: "nil context", input: "abcdefghijk", metadata: `{"id":"abcdefghijk"}`, wantErr: ErrInvalidInput},
		{name: "invalid video", ctx: context.Background(), input: "invalid", metadata: `{"id":"abcdefghijk"}`, wantErr: ErrInvalidInput},
		{name: "negative duration", ctx: context.Background(), input: "abcdefghijk", options: AudioOptions{MaxDuration: -1}, metadata: `{"id":"abcdefghijk"}`, wantErr: ErrInvalidInput},
		{name: "negative bytes", ctx: context.Background(), input: "abcdefghijk", options: AudioOptions{MaxBytes: -1}, metadata: `{"id":"abcdefghijk"}`, wantErr: ErrInvalidInput},
		{name: "missing dependency", ctx: context.Background(), input: "abcdefghijk", runnerErr: exec.ErrNotFound, wantErr: ErrMissingDependency},
		{name: "provider failure", ctx: context.Background(), input: "abcdefghijk", metadata: `{`, wantErr: ErrProviderFailure},
		{name: "canceled context", ctx: canceled, input: "abcdefghijk", metadata: `{"id":"abcdefghijk"}`, wantErr: context.Canceled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &publicAudioRunner{metadata: test.metadata, contents: test.contents, err: test.runnerErr}
			source := test.source
			if test.name != "nil receiver" {
				source = &YTDLPAudioSource{client: ytdlp.NewClient("yt-dlp", runner)}
			}
			audio, err := source.Acquire(test.ctx, test.input, test.options)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Acquire() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if err != nil {
				if audio.Data != nil || audio.Format != "" || audio.Duration != 0 || audio.Size != 0 {
					t.Fatalf("Acquire() audio = %#v after error, want zero value", audio)
				}
				return
			}
			data, readErr := io.ReadAll(audio.Data)
			if readErr != nil {
				t.Fatalf("ReadAll() error = %v", readErr)
			}
			if string(data) != test.contents || audio.Format != "webm" || audio.Duration != 2*time.Second || audio.Size != int64(len(test.contents)) {
				t.Fatalf("Acquire() audio = format %q, duration %s, size %d, data %q", audio.Format, audio.Duration, audio.Size, data)
			}
			if len(runner.args) != 2 || runner.args[0][len(runner.args[0])-1] != test.wantVideoID || runner.args[1][len(runner.args[1])-1] != test.wantVideoID {
				t.Fatalf("runner args = %q, want normalized video ID %q", runner.args, test.wantVideoID)
			}
			if closeErr := audio.Data.Close(); closeErr != nil {
				t.Fatalf("Close() error = %v", closeErr)
			}
			if _, statErr := os.Stat(runner.directory); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("temporary directory cleanup error = %v", statErr)
			}
		})
	}
}
