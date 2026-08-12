package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

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
		{name: "missing video", wantCode: 2, wantStderr: "usage:"},
		{name: "multiple videos", args: []string{"dQw4w9WgXcQ", "aqz-KE-bpKQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "unsupported format", args: []string{"--format", "xml", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
		{name: "zero timeout", args: []string{"--timeout", "0s", "dQw4w9WgXcQ"}, wantCode: 2, wantStderr: "usage:"},
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
