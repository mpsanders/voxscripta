package transcript_test

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	transcript "github.com/mpsanders/VoxScripta"
)

// TestYTDLPIntegration exercises the public API against the validated live
// matrix. It is opt-in because YouTube, network access, and yt-dlp availability
// are external mutable dependencies.
func TestYTDLPIntegration(t *testing.T) {
	if os.Getenv("VOXSCRIPTA_YTDLP_INTEGRATION") != "1" {
		t.Skip("set VOXSCRIPTA_YTDLP_INTEGRATION=1 to run live yt-dlp tests")
	}
	client, err := transcript.New()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		videoID    string
		options    transcript.Options
		cancel     bool
		wantSource transcript.SourceKind
		wantLang   string
		wantErr    error
	}{
		{name: "manual English captions", videoID: "O8G5Mkzhe4s", options: transcript.Options{Languages: []string{"en-US"}}, wantSource: transcript.SourceManual, wantLang: "en-US"},
		{name: "automatic Spanish captions", videoID: "4IVomi9s4BA", options: transcript.Options{Languages: []string{"es"}, AllowAutomatic: true}, wantSource: transcript.SourceAutomatic, wantLang: "es"},
		{name: "multilingual manual French", videoID: "W01c2-2NubU", options: transcript.Options{Languages: []string{"fr"}}, wantSource: transcript.SourceManual, wantLang: "fr"},
		{name: "no captions", videoID: "aqz-KE-bpKQ", options: transcript.Options{AllowAutomatic: true}, wantErr: transcript.ErrTranscriptUnavailable},
		{name: "canceled acquisition", videoID: "O8G5Mkzhe4s", cancel: true, wantErr: context.Canceled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			if test.cancel {
				cancel()
			} else {
				defer cancel()
			}
			result, err := client.Get(ctx, test.videoID, test.options)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Get() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if test.wantErr == nil {
				if result.Source != test.wantSource || result.Language.Code != test.wantLang || len(result.Segments) == 0 {
					t.Fatalf("Get() result source/language/segments = %q/%q/%d", result.Source, result.Language.Code, len(result.Segments))
				}
			}
		})
	}
}

// TestYTDLPAudioSourceIntegration exercises live audio acquisition and its
// duration, file-size, validation, cancellation, and ownership safeguards.
func TestYTDLPAudioSourceIntegration(t *testing.T) {
	if os.Getenv("VOXSCRIPTA_YTDLP_INTEGRATION") != "1" {
		t.Skip("set VOXSCRIPTA_YTDLP_INTEGRATION=1 to run live yt-dlp tests")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name      string
		source    *transcript.YTDLPAudioSource
		videoID   string
		options   transcript.AudioOptions
		wantErr   error
		cancel    bool
		wantAudio bool
	}{
		{name: "downloads short audio", source: transcript.NewYTDLPAudioSource(""), videoID: "O8G5Mkzhe4s", options: transcript.AudioOptions{MaxDuration: 2 * time.Minute, MaxBytes: 10 << 20}, wantAudio: true},
		{name: "rejects duration before download", source: transcript.NewYTDLPAudioSource(""), videoID: "O8G5Mkzhe4s", options: transcript.AudioOptions{MaxDuration: time.Second}, wantErr: transcript.ErrLimitExceeded},
		{name: "rejects known oversized download", source: transcript.NewYTDLPAudioSource(""), videoID: "O8G5Mkzhe4s", options: transcript.AudioOptions{MaxBytes: 1}, wantErr: transcript.ErrLimitExceeded},
		{name: "honors canceled context", source: transcript.NewYTDLPAudioSource(""), videoID: "O8G5Mkzhe4s", cancel: true, wantErr: context.Canceled},
		{name: "rejects negative limit", source: transcript.NewYTDLPAudioSource(""), videoID: "O8G5Mkzhe4s", options: transcript.AudioOptions{MaxBytes: -1}, wantErr: transcript.ErrInvalidInput},
		{name: "rejects nil source", videoID: "O8G5Mkzhe4s", wantErr: transcript.ErrInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, timeoutCancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer timeoutCancel()
			if test.cancel {
				ctx = canceled
			}
			audio, err := test.source.Acquire(ctx, test.videoID, test.options)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Acquire() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if err != nil {
				return
			}
			if test.wantAudio && (audio.Format == "" || audio.Size <= 0 || audio.Size > test.options.MaxBytes || audio.Duration <= 0 || audio.Duration > test.options.MaxDuration) {
				t.Fatalf("Acquire() format/size/duration = %q/%d/%s", audio.Format, audio.Size, audio.Duration)
			}
			buffer := make([]byte, 1)
			if _, readErr := io.ReadFull(audio.Data, buffer); readErr != nil {
				t.Fatalf("read acquired audio: %v", readErr)
			}
			if closeErr := audio.Data.Close(); closeErr != nil {
				t.Fatalf("close acquired audio: %v", closeErr)
			}
		})
	}
}
