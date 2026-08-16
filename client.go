package transcript

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/mpsanders/voxscripta/internal/ytdlp"
)

// Client coordinates input validation and transcript acquisition through a
// configured Provider. A Client is safe for concurrent use when its provider
// is safe for concurrent use.
type Client struct {
	provider Provider
}

// ClientOption configures a Client created by New.
type ClientOption func(*clientConfig) error

type clientConfig struct {
	provider    Provider
	ytdlpPath   string
	ytdlpRunner ytdlp.CommandRunner
}

// New constructs a transcript acquisition client. By default it uses the
// yt-dlp executable found on PATH. Options may select an explicit executable
// path or replace the acquisition provider.
func New(options ...ClientOption) (*Client, error) {
	configuration := clientConfig{ytdlpPath: "yt-dlp"}
	for index, option := range options {
		if option == nil {
			return nil, fmt.Errorf("%w: client option %d is nil", ErrInvalidInput, index)
		}
		if err := option(&configuration); err != nil {
			return nil, err
		}
	}
	if configuration.provider == nil {
		configuration.provider = &ytdlpProvider{client: ytdlp.NewClient(configuration.ytdlpPath, configuration.ytdlpRunner)}
	}
	return &Client{provider: configuration.provider}, nil
}

// WithYTDLPPath configures the executable name or path used by the default
// yt-dlp provider. Path must not be empty. This option cannot be combined with
// WithProvider because a custom provider does not use yt-dlp configuration.
func WithYTDLPPath(path string) ClientOption {
	return func(configuration *clientConfig) error {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("%w: yt-dlp path must not be empty", ErrInvalidInput)
		}
		if configuration.provider != nil {
			return fmt.Errorf("%w: yt-dlp path cannot be combined with a custom provider", ErrInvalidInput)
		}
		configuration.ytdlpPath = path
		return nil
	}
}

// WithProvider replaces the default yt-dlp provider with provider. The caller
// retains ownership of provider and is responsible for its concurrency safety.
func WithProvider(provider Provider) ClientOption {
	return func(configuration *clientConfig) error {
		if provider == nil {
			return fmt.Errorf("%w: provider must not be nil", ErrInvalidInput)
		}
		if configuration.ytdlpPath != "yt-dlp" || configuration.ytdlpRunner != nil {
			return fmt.Errorf("%w: custom provider cannot be combined with yt-dlp configuration", ErrInvalidInput)
		}
		configuration.provider = provider
		return nil
	}
}

// Get validates input, extracts its YouTube video ID, and asks the configured
// provider to acquire a normalized transcript using options.
func (c *Client) Get(ctx context.Context, input string, options Options) (Transcript, error) {
	if c == nil || c.provider == nil {
		return Transcript{}, fmt.Errorf("%w: client is not configured", ErrInvalidInput)
	}
	if ctx == nil {
		return Transcript{}, fmt.Errorf("%w: context must not be nil", ErrInvalidInput)
	}
	videoID, err := ParseVideoID(input)
	if err != nil {
		return Transcript{}, err
	}
	result, err := c.provider.Get(ctx, videoID, options)
	if err != nil {
		return Transcript{}, err
	}
	if err := result.Validate(); err != nil {
		return Transcript{}, fmt.Errorf("%w: provider returned invalid transcript: %v", ErrProviderFailure, err)
	}
	return result, nil
}

type ytdlpProvider struct {
	client *ytdlp.Client
}

// Get acquires one selected yt-dlp caption track and normalizes it.
func (p *ytdlpProvider) Get(ctx context.Context, videoID string, options Options) (Transcript, error) {
	version, err := p.client.Version(ctx)
	if err != nil {
		return Transcript{}, classifyYTDLPError("check yt-dlp", err)
	}
	metadata, err := p.client.Inspect(ctx, videoID)
	if err != nil {
		return Transcript{}, classifyYTDLPError("inspect video", err)
	}
	track, found := ytdlp.SelectTrack(metadata.Tracks, metadata.OriginalLanguage, options.Languages, options.AllowAutomatic)
	if !found {
		return Transcript{}, fmt.Errorf("%w: no caption track matched the requested language and source policy", ErrTranscriptUnavailable)
	}
	contents, err := p.client.Retrieve(ctx, videoID, track)
	if err != nil {
		return Transcript{}, classifyYTDLPError("retrieve captions", err)
	}
	segments, err := ParseWebVTT(strings.NewReader(string(contents)))
	if err != nil {
		if errors.Is(err, ErrUnsupportedFormat) {
			return Transcript{}, err
		}
		return Transcript{}, fmt.Errorf("%w: parse retrieved WebVTT: %v", ErrProviderFailure, err)
	}
	source := SourceManual
	if track.Source == ytdlp.CaptionAutomatic {
		source = SourceAutomatic
	}
	return Transcript{
		VideoID:  videoID,
		Title:    metadata.Title,
		Language: Language{Code: track.Language, Name: track.Name},
		Source:   source,
		Provider: ProviderMetadata{Name: "yt-dlp", Version: version},
		Segments: segments,
	}, nil
}

// classifyYTDLPError maps provider-specific failures onto the public errors.
func classifyYTDLPError(operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, exec.ErrDot) {
		return fmt.Errorf("%w: %s: %v", ErrMissingDependency, operation, err)
	}
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %s: %v", ErrTranscriptUnavailable, operation, err)
	}
	return fmt.Errorf("%w: %s: %v", ErrProviderFailure, operation, err)
}
