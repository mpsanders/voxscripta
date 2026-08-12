package ytdlp

import (
	"errors"
	"strings"
	"testing"
)

// TestSafeDiagnostic verifies that subprocess diagnostics remain useful while
// credential-bearing and unbounded values are not exposed to callers.
func TestSafeDiagnostic(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		notContains []string
		wantMax     int
	}{
		{name: "empty", input: "", want: ""},
		{name: "ordinary error", input: "ERROR: video unavailable\n", want: "ERROR: video unavailable"},
		{name: "signed URL", input: "failed https://example.test/caption?sig=secret&expire=1", want: "failed [redacted-url]", notContains: []string{"secret", "example.test"}},
		{name: "cookie header", input: "Cookie: SID=secret next", want: "Cookie: [redacted] next", notContains: []string{"SID=secret"}},
		{name: "authorization header", input: "Authorization = Bearer-secret", want: "Authorization = [redacted]", notContains: []string{"Bearer-secret"}},
		{name: "proxy option", input: "proxy=http://user:pass@example.test/path", want: "proxy=[redacted]", notContains: []string{"user:pass"}},
		{name: "multiline whitespace", input: "first\r\n  second\tthird", want: "first second third"},
		{name: "bounded output", input: strings.Repeat("x", maxDiagnosticLength+100), wantMax: maxDiagnosticLength + len("...")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := safeDiagnostic(test.input)
			if test.want != "" || test.input == "" {
				if got != test.want {
					t.Fatalf("safeDiagnostic() = %q, want %q", got, test.want)
				}
			}
			for _, forbidden := range test.notContains {
				if strings.Contains(got, forbidden) {
					t.Errorf("safeDiagnostic() = %q, unexpectedly contains %q", got, forbidden)
				}
			}
			if test.wantMax > 0 && len(got) > test.wantMax {
				t.Errorf("safeDiagnostic() length = %d, want at most %d", len(got), test.wantMax)
			}
		})
	}
}

// TestCommandError verifies safe formatting and standard error unwrapping.
func TestCommandError(t *testing.T) {
	cause := errors.New("exit status 1")
	tests := []struct {
		name       string
		err        *CommandError
		want       string
		wantUnwrap error
	}{
		{name: "nil receiver", want: "yt-dlp command failed"},
		{name: "cause only", err: &CommandError{Cause: cause}, want: "yt-dlp command failed: exit status 1", wantUnwrap: cause},
		{name: "with diagnostic", err: &CommandError{Cause: cause, Diagnostic: "video unavailable"}, want: "yt-dlp command failed: exit status 1: video unavailable", wantUnwrap: cause},
		{name: "nil cause", err: &CommandError{}, want: "yt-dlp command failed: <nil>"},
		{name: "diagnostic without cause", err: &CommandError{Diagnostic: "failure"}, want: "yt-dlp command failed: <nil>: failure"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Fatalf("Error() = %q, want %q", got, test.want)
			}
			if test.wantUnwrap != nil && !errors.Is(test.err, test.wantUnwrap) {
				t.Errorf("errors.Is(_, %v) = false", test.wantUnwrap)
			}
		})
	}
}

// TestBoundedBuffer verifies that process output capture retains only a fixed
// prefix while always reporting complete writes to the subprocess.
func TestBoundedBuffer(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		writes []string
		want   string
	}{
		{name: "zero limit", writes: []string{"discard"}},
		{name: "empty write", limit: 4, writes: []string{""}},
		{name: "under limit", limit: 4, writes: []string{"abc"}, want: "abc"},
		{name: "exact limit", limit: 4, writes: []string{"abcd"}, want: "abcd"},
		{name: "single write truncated", limit: 4, writes: []string{"abcdef"}, want: "abcd"},
		{name: "multiple writes truncated", limit: 5, writes: []string{"ab", "cdef", "gh"}, want: "abcde"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buffer := newBoundedBuffer(test.limit)
			for _, value := range test.writes {
				written, err := buffer.Write([]byte(value))
				if err != nil || written != len(value) {
					t.Fatalf("Write(%q) = %d, %v", value, written, err)
				}
			}
			if buffer.String() != test.want || string(buffer.Bytes()) != test.want {
				t.Fatalf("buffer = %q, want %q", buffer.String(), test.want)
			}
		})
	}
}
