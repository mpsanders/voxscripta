package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	transcript "github.com/mpsanders/VoxScripta"
)

var errTestFailure = errors.New("test failure")

// TestRunWithoutAcquisition covers argument handling that does not invoke an
// external yt-dlp process.
func TestRunWithoutAcquisition(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "version", args: []string{"--version"}, wantCode: 0, wantStdout: "dev\n"},
		{name: "check with video", args: []string{"--check", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "check with zero timeout", args: []string{"--check", "--timeout", "0s"}, wantCode: 2, wantStderr: "usage:"},
		{name: "missing video", wantCode: 2, wantStderr: "usage:"},
		{name: "multiple videos", args: []string{"dQw4w9WgXcQ", "aqz-KE-bpKQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "unsupported format", args: []string{"--format", "xml", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "zero timeout", args: []string{"--timeout", "0s", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "negative audio duration", args: []string{"--max-audio-duration", "-1s", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "negative audio bytes", args: []string{"--max-audio-bytes", "-1", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "invalid duration", args: []string{"--timeout", "later", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "invalid value"},
		{name: "empty language", args: []string{"--language", "", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "language must not be empty"},
		{name: "invalid video", args: []string{"invalid"}, wantCode: 2, wantStderr: "invalid input"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(test.args, &stdout, &stderr)
			if code != test.wantCode {
				t.Fatalf("run() code = %d, want %d; stderr = %q", code, test.wantCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.wantStdout) {
				t.Errorf("stdout = %q, want substring %q", stdout.String(), test.wantStdout)
			}
			if !strings.Contains(stderr.String(), test.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), test.wantStderr)
			}
		})
	}
}

// TestNewClient verifies that local speech fallback is enabled only by an
// explicit model while preserving constructor validation.
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		maxDuration time.Duration
		maxBytes    int64
		wantErr     error
	}{
		{name: "caption only"},
		{name: "whitespace model remains caption only", model: " "},
		{name: "speech enabled", model: "model.bin", maxDuration: time.Hour, maxBytes: 10},
		{name: "zero speech limits", model: "model.bin"},
		{name: "negative duration accepted by construction and rejected on use", model: "model.bin", maxDuration: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := newClient("yt-dlp", "whisper-cli", test.model, "ffmpeg", test.maxDuration, test.maxBytes)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("newClient() error = %v, want %v", err, test.wantErr)
			}
			if err == nil && client == nil {
				t.Fatal("newClient() returned nil client")
			}
		})
	}
}

// TestNewJSONTranscript verifies the CLI's human-readable timestamp contract.
func TestNewJSONTranscript(t *testing.T) {
	tests := []struct {
		name      string
		segments  []transcript.Segment
		wantStart string
		wantEnd   string
		wantCount int
	}{
		{name: "nil segments", wantCount: 0},
		{name: "zero values", segments: []transcript.Segment{{}}, wantStart: "0s", wantEnd: "0s", wantCount: 1},
		{name: "whole seconds", segments: []transcript.Segment{{Start: time.Second, End: 2 * time.Second}}, wantStart: "1s", wantEnd: "2s", wantCount: 1},
		{name: "fractional seconds", segments: []transcript.Segment{{Start: 1250 * time.Millisecond, End: 2500 * time.Millisecond}}, wantStart: "1.25s", wantEnd: "2.5s", wantCount: 1},
		{name: "minutes", segments: []transcript.Segment{{Start: 2 * time.Minute, End: 2*time.Minute + 3*time.Second}}, wantStart: "2m0s", wantEnd: "2m3s", wantCount: 1},
		{name: "multiple segments", segments: []transcript.Segment{{End: time.Second}, {Start: time.Second, End: 2 * time.Second}}, wantStart: "0s", wantEnd: "1s", wantCount: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := newJSONTranscript(transcript.Transcript{Segments: test.segments})
			if len(got.Segments) != test.wantCount {
				t.Fatalf("len(Segments) = %d, want %d", len(got.Segments), test.wantCount)
			}
			if test.wantCount > 0 && (got.Segments[0].Start != test.wantStart || got.Segments[0].End != test.wantEnd) {
				t.Errorf("first segment = %#v, want start %q and end %q", got.Segments[0], test.wantStart, test.wantEnd)
			}
		})
	}
}

// TestExitCode verifies the stable mapping from public error categories.
func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", want: 1},
		{name: "invalid input", err: transcript.ErrInvalidInput, want: 2},
		{name: "missing dependency", err: transcript.ErrMissingDependency, want: 3},
		{name: "unavailable transcript", err: transcript.ErrTranscriptUnavailable, want: 4},
		{name: "uncategorized", err: errTestFailure, want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := exitCode(test.err); got != test.want {
				t.Fatalf("exitCode() = %d, want %d", got, test.want)
			}
		})
	}
}
