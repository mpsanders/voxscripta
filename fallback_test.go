package transcript

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type recordingProvider struct {
	result  Transcript
	err     error
	calls   int
	videoID string
	options Options
}

// Get records the request and returns the provider's configured result.
func (p *recordingProvider) Get(_ context.Context, videoID string, options Options) (Transcript, error) {
	p.calls++
	p.videoID = videoID
	p.options = options
	return p.result, p.err
}

// TestFallbackProviderGet verifies the explicit and deliberately narrow
// fallback policy across success, unavailable, and non-fallback failures.
func TestFallbackProviderGet(t *testing.T) {
	primaryResult := Transcript{VideoID: "primary001A"}
	fallbackResult := Transcript{VideoID: "fallback01A"}
	options := Options{Languages: []string{"en-AU", "en"}, AllowAutomatic: true}

	tests := []struct {
		name          string
		primary       Provider
		fallback      Provider
		want          Transcript
		wantErr       error
		fallbackCalls int
	}{
		{name: "primary success", primary: &recordingProvider{result: primaryResult}, fallback: &recordingProvider{result: fallbackResult}, want: primaryResult},
		{name: "unavailable invokes fallback", primary: &recordingProvider{err: ErrTranscriptUnavailable}, fallback: &recordingProvider{result: fallbackResult}, want: fallbackResult, fallbackCalls: 1},
		{name: "wrapped unavailable invokes fallback", primary: &recordingProvider{err: errors.Join(errors.New("captions absent"), ErrTranscriptUnavailable)}, fallback: &recordingProvider{result: fallbackResult}, want: fallbackResult, fallbackCalls: 1},
		{name: "provider failure does not fallback", primary: &recordingProvider{err: ErrProviderFailure}, fallback: &recordingProvider{result: fallbackResult}, wantErr: ErrProviderFailure},
		{name: "cancellation does not fallback", primary: &recordingProvider{err: context.Canceled}, fallback: &recordingProvider{result: fallbackResult}, wantErr: context.Canceled},
		{name: "missing dependency does not fallback", primary: &recordingProvider{err: ErrMissingDependency}, fallback: &recordingProvider{result: fallbackResult}, wantErr: ErrMissingDependency},
		{name: "nil primary is invalid", fallback: &recordingProvider{result: fallbackResult}, wantErr: ErrInvalidInput},
		{name: "nil fallback is invalid", primary: &recordingProvider{result: primaryResult}, wantErr: ErrInvalidInput},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := (FallbackProvider{Primary: test.primary, Fallback: test.fallback}).Get(context.Background(), "abcdefghijk", options)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Get() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Get() = %#v, want %#v", got, test.want)
			}
			if provider, ok := test.fallback.(*recordingProvider); ok {
				if provider.calls != test.fallbackCalls {
					t.Fatalf("fallback calls = %d, want %d", provider.calls, test.fallbackCalls)
				}
				if provider.calls > 0 && (provider.videoID != "abcdefghijk" || len(provider.options.Languages) != 2 || !provider.options.AllowAutomatic) {
					t.Fatalf("fallback request = (%q, %#v), want original request", provider.videoID, provider.options)
				}
			}
		})
	}
}
