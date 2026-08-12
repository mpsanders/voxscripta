package transcript_test

import (
	"context"
	"errors"
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
