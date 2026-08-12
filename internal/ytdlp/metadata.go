package ytdlp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// CaptionSource identifies whether yt-dlp reported a creator-provided or
// automatically generated caption track.
type CaptionSource uint8

const (
	// CaptionManual identifies a creator-provided caption track.
	CaptionManual CaptionSource = iota + 1
	// CaptionAutomatic identifies an automatically generated caption track.
	CaptionAutomatic
)

// CaptionTrack is the narrow internal representation used for selection and
// retrieval. URL is sensitive and must never be included in diagnostics.
type CaptionTrack struct {
	Language string
	Name     string
	Format   string
	URL      string
	Source   CaptionSource
}

// Metadata contains the video details and usable WebVTT caption tracks found
// during a yt-dlp inspection.
type Metadata struct {
	ID               string
	Title            string
	OriginalLanguage string
	Duration         time.Duration
	DurationKnown    bool
	IsLive           bool
	Tracks           []CaptionTrack
}

type rawMetadata struct {
	ID                string                        `json:"id"`
	Title             string                        `json:"title"`
	Language          string                        `json:"language"`
	Duration          *float64                      `json:"duration"`
	IsLive            bool                          `json:"is_live"`
	Subtitles         map[string][]rawCaptionFormat `json:"subtitles"`
	AutomaticCaptions map[string][]rawCaptionFormat `json:"automatic_captions"`
}

type rawCaptionFormat struct {
	Extension string `json:"ext"`
	Name      string `json:"name"`
	URL       string `json:"url"`
}

// Inspect requests one JSON metadata object for videoID and decodes only the
// fields required for caption selection. videoID must already be validated.
func (c *Client) Inspect(ctx context.Context, videoID string) (Metadata, error) {
	if c == nil || c.runner == nil {
		return Metadata{}, errors.New("yt-dlp client is not configured")
	}
	arguments := []string{ignoreConfigArgument, "--dump-single-json", "--skip-download", "--no-warnings", "--", videoID}
	result, err := c.runner.Run(ctx, c.executable, arguments...)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Metadata{}, ctxErr
		}
		return Metadata{}, fmt.Errorf("inspect video with yt-dlp: %w", err)
	}
	var raw rawMetadata
	if err := json.Unmarshal(result.Stdout, &raw); err != nil {
		return Metadata{}, fmt.Errorf("decode yt-dlp metadata: %w", err)
	}
	if strings.TrimSpace(raw.ID) == "" {
		return Metadata{}, errors.New("yt-dlp metadata is missing video ID")
	}
	duration, durationKnown, err := decodeDuration(raw.Duration)
	if err != nil {
		return Metadata{}, err
	}
	metadata := Metadata{
		ID: raw.ID, Title: raw.Title, OriginalLanguage: raw.Language,
		Duration: duration, DurationKnown: durationKnown, IsLive: raw.IsLive,
	}
	metadata.Tracks = appendTracks(metadata.Tracks, raw.Subtitles, CaptionManual)
	metadata.Tracks = appendTracks(metadata.Tracks, raw.AutomaticCaptions, CaptionAutomatic)
	return metadata, nil
}

// decodeDuration converts optional yt-dlp duration seconds without allowing
// negative, non-finite, or time.Duration-overflowing values into metadata.
func decodeDuration(seconds *float64) (time.Duration, bool, error) {
	if seconds == nil {
		return 0, false, nil
	}
	nanoseconds := *seconds * float64(time.Second)
	if math.IsNaN(*seconds) || math.IsInf(*seconds, 0) || *seconds < 0 || nanoseconds >= float64(math.MaxInt64) {
		return 0, false, fmt.Errorf("yt-dlp metadata has invalid duration %v", *seconds)
	}
	return time.Duration(nanoseconds), true, nil
}

// appendTracks adds one usable WebVTT format per language in deterministic
// language order. Non-WebVTT and URL-less formats are ignored.
func appendTracks(destination []CaptionTrack, formats map[string][]rawCaptionFormat, source CaptionSource) []CaptionTrack {
	languages := make([]string, 0, len(formats))
	for language := range formats {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	for _, language := range languages {
		for _, format := range formats[language] {
			if format.Extension == "vtt" && strings.TrimSpace(format.URL) != "" {
				destination = append(destination, CaptionTrack{
					Language: language, Name: format.Name, Format: format.Extension,
					URL: format.URL, Source: source,
				})
				break
			}
		}
	}
	return destination
}
