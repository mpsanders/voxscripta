package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrAudioLimit indicates that configured duration or size limits rejected an
// audio acquisition.
var ErrAudioLimit = errors.New("audio limit exceeded")

// AudioArtifact describes a downloaded audio file. Closing Data removes the
// isolated temporary directory containing the file.
type AudioArtifact struct {
	Data     io.ReadCloser
	Format   string
	Duration time.Duration
	Size     int64
}

// AcquireAudio inspects videoID and downloads its best available audio-only
// stream.
// maxDuration and maxBytes are optional limits; zero disables the corresponding
// limit. The caller owns the returned Data and must close it to remove the
// downloaded artifact.
func (c *Client) AcquireAudio(ctx context.Context, videoID string, maxDuration time.Duration, maxBytes int64) (artifact AudioArtifact, returnedErr error) {
	if c == nil || c.runner == nil {
		return AudioArtifact{}, errors.New("yt-dlp client is not configured")
	}
	if ctx == nil {
		return AudioArtifact{}, errors.New("context must not be nil")
	}
	if strings.TrimSpace(videoID) == "" {
		return AudioArtifact{}, errors.New("video ID must not be empty")
	}
	if maxDuration < 0 || maxBytes < 0 {
		return AudioArtifact{}, errors.New("audio limits must not be negative")
	}

	metadata, err := c.Inspect(ctx, videoID)
	if err != nil {
		return AudioArtifact{}, fmt.Errorf("inspect audio metadata: %w", err)
	}
	if metadata.Duration < 0 {
		return AudioArtifact{}, errors.New("yt-dlp reported a negative video duration")
	}
	if maxDuration > 0 {
		if metadata.IsLive || !metadata.DurationKnown {
			return AudioArtifact{}, fmt.Errorf("%w: video duration is unknown or live and cannot be checked against %s", ErrAudioLimit, maxDuration)
		}
		if metadata.Duration > maxDuration {
			return AudioArtifact{}, fmt.Errorf("%w: video duration %s exceeds limit %s", ErrAudioLimit, metadata.Duration, maxDuration)
		}
	}

	directory, err := os.MkdirTemp("", "voxscripta-audio-*")
	if err != nil {
		return AudioArtifact{}, fmt.Errorf("create audio temporary directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(directory); err != nil {
				returnedErr = errors.Join(returnedErr, fmt.Errorf("remove audio temporary directory: %w", err))
			}
		}
	}()

	arguments := []string{ignoreConfigArgument,
		"--no-playlist", "--quiet", "--no-progress", "--no-warnings",
		"--format", "bestaudio", "--output", filepath.Join(directory, "audio.%(ext)s"),
	}
	if maxBytes > 0 {
		arguments = append(arguments, "--max-filesize", strconv.FormatInt(maxBytes, 10))
	}
	arguments = append(arguments, "--", videoID)
	if _, err := c.runner.Run(ctx, c.executable, arguments...); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return AudioArtifact{}, ctxErr
		}
		if maxBytes > 0 && isFileSizeRejection(err) {
			return AudioArtifact{}, fmt.Errorf("%w: yt-dlp rejected audio larger than %d bytes", ErrAudioLimit, maxBytes)
		}
		return AudioArtifact{}, fmt.Errorf("retrieve audio with yt-dlp: %w", err)
	}

	path, info, err := singleAudioPath(directory)
	if err != nil {
		if maxBytes > 0 && errors.Is(err, errNoAudioFile) {
			return AudioArtifact{}, fmt.Errorf("%w: yt-dlp produced no audio file within the %d-byte limit", ErrAudioLimit, maxBytes)
		}
		return AudioArtifact{}, err
	}
	if info.Size() == 0 {
		return AudioArtifact{}, errors.New("yt-dlp produced an empty audio file")
	}
	if maxBytes > 0 && info.Size() > maxBytes {
		return AudioArtifact{}, fmt.Errorf("%w: downloaded audio size %d exceeds limit %d", ErrAudioLimit, info.Size(), maxBytes)
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if format == "" {
		return AudioArtifact{}, errors.New("yt-dlp produced an audio file without a format extension")
	}
	file, err := os.Open(path)
	if err != nil {
		return AudioArtifact{}, fmt.Errorf("open downloaded audio: %w", err)
	}
	cleanup = false
	return AudioArtifact{
		Data: &cleanupFile{File: file, directory: directory}, Format: format,
		Duration: metadata.Duration, Size: info.Size(),
	}, nil
}

var errNoAudioFile = errors.New("yt-dlp produced no audio file")

// isFileSizeRejection recognizes the bounded diagnostic emitted by yt-dlp
// when it can reject a download using known file-size metadata.
func isFileSizeRejection(err error) bool {
	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		return false
	}
	diagnostic := strings.ToLower(commandErr.Diagnostic)
	return strings.Contains(diagnostic, "max-filesize") ||
		(strings.Contains(diagnostic, "larger than") && strings.Contains(diagnostic, "file"))
}

// singleAudioPath returns the only regular downloaded artifact in directory.
func singleAudioPath(directory string) (string, os.FileInfo, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", nil, fmt.Errorf("inspect downloaded audio output: %w", err)
	}
	var path string
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			if path != "" {
				return "", nil, errors.New("yt-dlp produced multiple audio files")
			}
			path = filepath.Join(directory, entry.Name())
		}
	}
	if path == "" {
		return "", nil, errNoAudioFile
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("inspect downloaded audio: %w", err)
	}
	return path, info, nil
}

type cleanupFile struct {
	*os.File
	directory string
	once      sync.Once
	err       error
}

// Close closes the audio file and removes its containing temporary directory.
func (f *cleanupFile) Close() error {
	f.once.Do(func() {
		closeErr := f.File.Close()
		removeErr := os.RemoveAll(f.directory)
		f.err = errors.Join(closeErr, removeErr)
	})
	return f.err
}
