package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Retrieve downloads the selected WebVTT caption track into an isolated
// temporary directory and returns its contents. The videoID identifies the
// already-inspected video, and track must be the exact track selected from its
// metadata. Temporary output is removed before Retrieve returns.
func (c *Client) Retrieve(ctx context.Context, videoID string, track CaptionTrack) ([]byte, error) {
	if c == nil || c.runner == nil {
		return nil, errors.New("yt-dlp client is not configured")
	}
	if strings.TrimSpace(videoID) == "" {
		return nil, errors.New("video ID must not be empty")
	}
	if strings.TrimSpace(track.Language) == "" {
		return nil, errors.New("caption language must not be empty")
	}
	if track.Format != "vtt" {
		return nil, fmt.Errorf("unsupported caption format %q", track.Format)
	}

	temporaryDirectory, err := os.MkdirTemp("", "voxscripta-caption-*")
	if err != nil {
		return nil, fmt.Errorf("create caption temporary directory: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	outputTemplate := filepath.Join(temporaryDirectory, "caption.%(ext)s")
	arguments := []string{
		"--skip-download", "--no-warnings", "--sub-langs", track.Language,
		"--sub-format", "vtt", "--output", outputTemplate,
	}
	switch track.Source {
	case CaptionManual:
		arguments = append(arguments, "--write-subs")
	case CaptionAutomatic:
		arguments = append(arguments, "--write-auto-subs")
	default:
		return nil, fmt.Errorf("unsupported caption source %d", track.Source)
	}
	arguments = append(arguments, "--", videoID)

	_, err = c.runner.Run(ctx, c.executable, arguments...)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("retrieve caption with yt-dlp: %w", err)
	}

	captionPath, err := singleWebVTTPath(temporaryDirectory)
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(captionPath)
	if err != nil {
		return nil, fmt.Errorf("read retrieved caption: %w", err)
	}
	return contents, nil
}

// singleWebVTTPath returns the only regular WebVTT file directly contained in
// directory. It rejects missing or ambiguous output instead of guessing which
// generated file belongs to the selected caption track.
func singleWebVTTPath(directory string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("inspect retrieved caption output: %w", err)
	}
	var captionPath string
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".vtt") {
			continue
		}
		if captionPath != "" {
			return "", errors.New("yt-dlp produced multiple WebVTT caption files")
		}
		captionPath = filepath.Join(directory, entry.Name())
	}
	if captionPath == "" {
		return "", errors.New("yt-dlp produced no WebVTT caption file")
	}
	return captionPath, nil
}
