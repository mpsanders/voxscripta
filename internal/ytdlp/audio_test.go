package ytdlp

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type audioRunner struct {
	metadata       string
	files          map[string]string
	downloadErr    error
	cancelDownload context.CancelFunc
	args           [][]string
	directory      string
	directoryAlive bool
}

// Run returns metadata for inspection and creates configured download files.
func (r *audioRunner) Run(ctx context.Context, _ string, args ...string) (CommandResult, error) {
	r.args = append(r.args, append([]string(nil), args...))
	if err := ctx.Err(); err != nil {
		return CommandResult{}, err
	}
	if containsArgument(args, "--dump-single-json") {
		return CommandResult{Stdout: []byte(r.metadata)}, nil
	}
	for index, argument := range args {
		if argument != "--output" || index+1 >= len(args) {
			continue
		}
		r.directory = filepath.Dir(args[index+1])
		_, statErr := os.Stat(r.directory)
		r.directoryAlive = statErr == nil
		for name, contents := range r.files {
			path := filepath.Join(r.directory, name)
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				return CommandResult{}, err
			}
		}
	}
	if r.cancelDownload != nil {
		r.cancelDownload()
		<-ctx.Done()
		return CommandResult{}, ctx.Err()
	}
	return CommandResult{}, r.downloadErr
}

// TestClientAcquireAudio verifies preflight limits, the exact download
// contract, output validation, mid-download cancellation, and cleanup.
func TestClientAcquireAudio(t *testing.T) {
	tests := []struct {
		name             string
		metadata         string
		files            map[string]string
		downloadErr      error
		maxDuration      time.Duration
		maxBytes         int64
		videoID          string
		nilContext       bool
		preCanceled      bool
		cancelDownload   bool
		nilClient        bool
		wantErr          error
		wantContains     string
		wantFormat       string
		wantContents     string
		wantDuration     time.Duration
		wantDownload     bool
		wantMetadataCall bool
	}{
		{name: "successful webm", metadata: `{"id":"abcdefghijk","duration":30}`, files: map[string]string{"audio.webm": "data"}, maxDuration: time.Minute, maxBytes: 10, wantFormat: "webm", wantContents: "data", wantDuration: 30 * time.Second, wantDownload: true, wantMetadataCall: true},
		{name: "zero limits allow unknown duration", metadata: `{"id":"abcdefghijk"}`, files: map[string]string{"audio.M4A": "x"}, wantFormat: "m4a", wantContents: "x", wantDownload: true, wantMetadataCall: true},
		{name: "duration equal to limit", metadata: `{"id":"abcdefghijk","duration":60}`, files: map[string]string{"audio.opus": "x"}, maxDuration: time.Minute, wantFormat: "opus", wantContents: "x", wantDuration: time.Minute, wantDownload: true, wantMetadataCall: true},
		{name: "duration rejected before download", metadata: `{"id":"abcdefghijk","duration":61}`, maxDuration: time.Minute, wantErr: ErrAudioLimit, wantContains: "exceeds", wantMetadataCall: true},
		{name: "unknown duration rejected with limit", metadata: `{"id":"abcdefghijk"}`, maxDuration: time.Minute, wantErr: ErrAudioLimit, wantContains: "unknown or live", wantMetadataCall: true},
		{name: "live input rejected with limit", metadata: `{"id":"abcdefghijk","duration":30,"is_live":true}`, maxDuration: time.Minute, wantErr: ErrAudioLimit, wantContains: "unknown or live", wantMetadataCall: true},
		{name: "final size rejected", metadata: `{"id":"abcdefghijk","duration":1}`, files: map[string]string{"audio.opus": "oversize"}, maxBytes: 4, wantErr: ErrAudioLimit, wantContains: "size", wantDownload: true, wantMetadataCall: true},
		{name: "empty artifact rejected", metadata: `{"id":"abcdefghijk"}`, files: map[string]string{"audio.m4a": ""}, wantContains: "empty audio", wantDownload: true, wantMetadataCall: true},
		{name: "missing format extension rejected", metadata: `{"id":"abcdefghijk"}`, files: map[string]string{"audio": "x"}, wantContains: "without a format extension", wantDownload: true, wantMetadataCall: true},
		{name: "missing output under size limit", metadata: `{"id":"abcdefghijk"}`, maxBytes: 4, wantErr: ErrAudioLimit, wantContains: "no audio file", wantDownload: true, wantMetadataCall: true},
		{name: "missing output without size limit", metadata: `{"id":"abcdefghijk"}`, wantContains: "no audio file", wantDownload: true, wantMetadataCall: true},
		{name: "multiple outputs", metadata: `{"id":"abcdefghijk"}`, files: map[string]string{"one.webm": "1", "two.m4a": "2"}, wantContains: "multiple audio", wantDownload: true, wantMetadataCall: true},
		{name: "size diagnostic maps limit", metadata: `{"id":"abcdefghijk"}`, downloadErr: &CommandError{Cause: errors.New("exit status 1"), Diagnostic: "File is larger than max-filesize"}, maxBytes: 4, wantErr: ErrAudioLimit, wantDownload: true, wantMetadataCall: true},
		{name: "process failure cleans partial file", metadata: `{"id":"abcdefghijk"}`, files: map[string]string{"audio.webm.part": "partial"}, downloadErr: errors.New("exit status 1"), wantContains: "retrieve audio", wantDownload: true, wantMetadataCall: true},
		{name: "mid-download cancellation cleans partial file", metadata: `{"id":"abcdefghijk"}`, files: map[string]string{"audio.webm.part": "partial"}, cancelDownload: true, wantErr: context.Canceled, wantDownload: true, wantMetadataCall: true},
		{name: "pre-canceled context", metadata: `{"id":"abcdefghijk"}`, preCanceled: true, wantErr: context.Canceled, wantMetadataCall: true},
		{name: "negative duration limit", metadata: `{"id":"abcdefghijk"}`, maxDuration: -1, wantContains: "negative"},
		{name: "negative byte limit", metadata: `{"id":"abcdefghijk"}`, maxBytes: -1, wantContains: "negative"},
		{name: "empty video ID", metadata: `{"id":"abcdefghijk"}`, videoID: " ", wantContains: "empty"},
		{name: "nil context", metadata: `{"id":"abcdefghijk"}`, nilContext: true, wantContains: "nil"},
		{name: "nil client", metadata: `{"id":"abcdefghijk"}`, nilClient: true, wantContains: "not configured"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			var cancel context.CancelFunc
			if test.preCanceled || test.cancelDownload {
				ctx, cancel = context.WithCancel(ctx)
				if test.preCanceled {
					cancel()
				}
				defer cancel()
			}
			if test.nilContext {
				ctx = nil
			}
			runner := &audioRunner{metadata: test.metadata, files: test.files, downloadErr: test.downloadErr}
			if test.cancelDownload {
				runner.cancelDownload = cancel
			}
			client := NewClient("yt-dlp", runner)
			if test.nilClient {
				client = nil
			}
			videoID := test.videoID
			if videoID == "" {
				videoID = "abcdefghijk"
			}
			artifact, err := client.AcquireAudio(ctx, videoID, test.maxDuration, test.maxBytes)
			if test.wantErr == nil && test.wantContains == "" && err != nil {
				t.Fatalf("AcquireAudio() error = %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("AcquireAudio() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if test.wantContains != "" && (err == nil || !strings.Contains(err.Error(), test.wantContains)) {
				t.Fatalf("AcquireAudio() error = %v, want substring %q", err, test.wantContains)
			}
			if err != nil && (artifact.Data != nil || artifact.Format != "" || artifact.Duration != 0 || artifact.Size != 0) {
				t.Fatalf("AcquireAudio() artifact = %#v after error, want zero value", artifact)
			}
			if err == nil {
				contents, readErr := io.ReadAll(artifact.Data)
				if readErr != nil {
					t.Fatalf("ReadAll() error = %v", readErr)
				}
				if artifact.Format != test.wantFormat || string(contents) != test.wantContents || artifact.Size != int64(len(test.wantContents)) || artifact.Duration != test.wantDuration {
					t.Errorf("artifact format/contents/size/duration = %q/%q/%d/%s", artifact.Format, contents, artifact.Size, artifact.Duration)
				}
				if _, statErr := os.Stat(runner.directory); statErr != nil {
					t.Errorf("owned directory stat error = %v", statErr)
				}
				if closeErr := artifact.Data.Close(); closeErr != nil {
					t.Fatalf("Close() error = %v", closeErr)
				}
				if closeErr := artifact.Data.Close(); closeErr != nil {
					t.Fatalf("second Close() error = %v", closeErr)
				}
			}
			if runner.directory != "" {
				if !runner.directoryAlive {
					t.Error("temporary directory was absent during download")
				}
				if _, statErr := os.Stat(runner.directory); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("temporary directory cleanup error = %v, want os.ErrNotExist", statErr)
				}
			}
			gotDownload := runner.directory != ""
			if gotDownload != test.wantDownload {
				t.Errorf("download attempted = %v, want %v", gotDownload, test.wantDownload)
			}
			if test.wantMetadataCall && len(runner.args) > 0 {
				wantInspect := []string{"--ignore-config", "--dump-single-json", "--skip-download", "--no-warnings", "--", "abcdefghijk"}
				if !reflect.DeepEqual(runner.args[0], wantInspect) {
					t.Errorf("inspect args = %q, want %q", runner.args[0], wantInspect)
				}
			}
			if test.wantDownload {
				wantDownload := []string{"--ignore-config", "--no-playlist", "--quiet", "--no-progress", "--no-warnings", "--format", "bestaudio", "--output", filepath.Join(runner.directory, "audio.%(ext)s")}
				if test.maxBytes > 0 {
					wantDownload = append(wantDownload, "--max-filesize", strconv.FormatInt(test.maxBytes, 10))
				}
				wantDownload = append(wantDownload, "--", "abcdefghijk")
				if len(runner.args) != 2 || !reflect.DeepEqual(runner.args[1], wantDownload) {
					t.Errorf("download calls/args = %d/%q, want 2/%q", len(runner.args), runner.args, wantDownload)
				}
			}
		})
	}
}
