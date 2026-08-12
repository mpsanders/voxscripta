package ytdlp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestClientInspect(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	validJSON := `{"id":"abcdefghijk","title":"Example","language":"en-US","duration":1.25,"is_live":true,"unknown":true,"subtitles":{"fr":[{"ext":"srv3","url":"ignored"},{"ext":"vtt","name":"French","url":"https://secret/manual"}],"en":[{"ext":"vtt","name":"English","url":"https://secret/en"}]},"automatic_captions":{"de":[{"ext":"vtt","name":"German","url":"https://secret/auto"}]}}`
	tests := []struct {
		name       string
		ctx        context.Context
		output     string
		runErr     error
		want       Metadata
		wantErr    bool
		wantCancel bool
	}{
		{name: "decodes required fields and ignores additions", ctx: context.Background(), output: validJSON, want: Metadata{ID: "abcdefghijk", Title: "Example", OriginalLanguage: "en-US", Duration: 1250 * time.Millisecond, DurationKnown: true, IsLive: true, Tracks: []CaptionTrack{{Language: "en", Name: "English", Format: "vtt", URL: "https://secret/en", Source: CaptionManual}, {Language: "fr", Name: "French", Format: "vtt", URL: "https://secret/manual", Source: CaptionManual}, {Language: "de", Name: "German", Format: "vtt", URL: "https://secret/auto", Source: CaptionAutomatic}}}},
		{name: "empty caption maps", ctx: context.Background(), output: `{"id":"abcdefghijk","subtitles":null,"automatic_captions":{}}`, want: Metadata{ID: "abcdefghijk"}},
		{name: "known zero duration", ctx: context.Background(), output: `{"id":"abcdefghijk","duration":0}`, want: Metadata{ID: "abcdefghijk", DurationKnown: true}},
		{name: "ignores unusable formats", ctx: context.Background(), output: `{"id":"abcdefghijk","subtitles":{"en":[{"ext":"json3","url":"x"},{"ext":"vtt","url":""}]}}`, want: Metadata{ID: "abcdefghijk"}},
		{name: "negative duration", ctx: context.Background(), output: `{"id":"abcdefghijk","duration":-1}`, wantErr: true},
		{name: "overflowing duration", ctx: context.Background(), output: `{"id":"abcdefghijk","duration":1e20}`, wantErr: true},
		{name: "malformed JSON", ctx: context.Background(), output: `{`, wantErr: true},
		{name: "zero output", ctx: context.Background(), wantErr: true},
		{name: "missing ID", ctx: context.Background(), output: `{}`, wantErr: true},
		{name: "process failure", ctx: context.Background(), runErr: errors.New("exit status 1"), wantErr: true},
		{name: "cancellation", ctx: canceled, runErr: errors.New("killed"), wantErr: true, wantCancel: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{result: CommandResult{Stdout: []byte(tt.output)}, err: tt.runErr}
			got, err := NewClient("yt-dlp", runner).Inspect(tt.ctx, "abcdefghijk")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Inspect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantCancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("Inspect() error = %v, want context.Canceled", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Inspect() = %#v, want %#v", got, tt.want)
			}
			wantArgs := []string{"--ignore-config", "--dump-single-json", "--skip-download", "--no-warnings", "--", "abcdefghijk"}
			if !reflect.DeepEqual(runner.args, wantArgs) {
				t.Errorf("runner args = %q, want %q", runner.args, wantArgs)
			}
		})
	}
}
