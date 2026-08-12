package transcript

import (
	"context"
	"errors"
	"testing"
	"time"
)

type providerFunc func(context.Context, string, Options) (Transcript, error)

// Get calls the provider function with the acquisition parameters.
func (provider providerFunc) Get(ctx context.Context, videoID string, options Options) (Transcript, error) {
	return provider(ctx, videoID, options)
}

func TestNew(t *testing.T) {
	provider := providerFunc(func(context.Context, string, Options) (Transcript, error) {
		return validClientTranscript(), nil
	})
	tests := []struct {
		name    string
		options []ClientOption
		wantErr error
	}{
		{name: "default client"},
		{name: "explicit executable", options: []ClientOption{WithYTDLPPath("custom-yt-dlp")}},
		{name: "custom provider", options: []ClientOption{WithProvider(provider)}},
		{name: "nil option", options: []ClientOption{nil}, wantErr: ErrInvalidInput},
		{name: "empty executable", options: []ClientOption{WithYTDLPPath("  ")}, wantErr: ErrInvalidInput},
		{name: "nil provider", options: []ClientOption{WithProvider(nil)}, wantErr: ErrInvalidInput},
		{name: "provider then executable", options: []ClientOption{WithProvider(provider), WithYTDLPPath("tool")}, wantErr: ErrInvalidInput},
		{name: "executable then provider", options: []ClientOption{WithYTDLPPath("tool"), WithProvider(provider)}, wantErr: ErrInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := New(test.options...)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("New() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if test.wantErr == nil && (client == nil || client.provider == nil) {
				t.Fatal("New() returned an unconfigured client")
			}
		})
	}
}

func TestClientGet(t *testing.T) {
	providerError := errors.New("provider exploded")
	tests := []struct {
		name        string
		client      *Client
		ctx         context.Context
		input       string
		result      Transcript
		providerErr error
		wantErr     error
	}{
		{name: "raw video ID", ctx: context.Background(), input: "dQw4w9WgXcQ", result: validClientTranscript()},
		{name: "video URL", ctx: context.Background(), input: "https://youtu.be/dQw4w9WgXcQ", result: validClientTranscript()},
		{name: "nil context", ctx: nil, input: "dQw4w9WgXcQ", wantErr: ErrInvalidInput},
		{name: "invalid input", ctx: context.Background(), input: "not-video", wantErr: ErrInvalidInput},
		{name: "nil client", client: &Client{}, ctx: context.Background(), input: "dQw4w9WgXcQ", wantErr: ErrInvalidInput},
		{name: "provider error", ctx: context.Background(), input: "dQw4w9WgXcQ", providerErr: providerError, wantErr: providerError},
		{name: "invalid provider result", ctx: context.Background(), input: "dQw4w9WgXcQ", result: Transcript{}, wantErr: ErrProviderFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := test.client
			if client == nil {
				client = &Client{provider: providerFunc(func(_ context.Context, videoID string, _ Options) (Transcript, error) {
					if videoID != "dQw4w9WgXcQ" {
						t.Fatalf("provider videoID = %q", videoID)
					}
					return test.result, test.providerErr
				})}
			}
			result, err := client.Get(test.ctx, test.input, Options{})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Get() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if test.wantErr == nil && result.VideoID != "dQw4w9WgXcQ" {
				t.Fatalf("Get() VideoID = %q", result.VideoID)
			}
		})
	}
}

// validClientTranscript returns a minimal normalized transcript for client tests.
func validClientTranscript() Transcript {
	return Transcript{
		VideoID: "dQw4w9WgXcQ", Language: Language{Code: "en"}, Source: SourceManual,
		Segments: []Segment{{Start: 0, End: time.Second, Text: "hello"}},
	}
}
