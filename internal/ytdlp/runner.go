package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const maxDiagnosticLength = 2048

var (
	urlDiagnosticPattern    = regexp.MustCompile(`https?://[^\s"']+`)
	secretDiagnosticPattern = regexp.MustCompile(`(?i)(authorization|cookie|proxy|token)(\s*[:=]\s*)([^\s,;]+)`)
)

// CommandResult contains the outputs needed to interpret a completed yt-dlp
// command.
type CommandResult struct {
	Stdout []byte
	Stderr []byte
}

// CommandRunner executes an executable directly with the supplied arguments.
// Implementations must not invoke a command shell and must honor ctx.
type CommandRunner interface {
	Run(ctx context.Context, executable string, args ...string) (CommandResult, error)
}

// ExecRunner executes commands with os/exec and separate stdout and stderr
// buffers. The executable parameter is a path or name resolved by os/exec.
type ExecRunner struct{}

// CommandError describes a failed direct process invocation without retaining
// command arguments, which can contain credentials. Diagnostic contains a
// bounded, redacted form of stderr and Cause retains the process error for
// errors.Is and errors.As checks.
type CommandError struct {
	Cause      error
	Diagnostic string
}

// Error renders the process failure and its safe diagnostic, when available.
func (e *CommandError) Error() string {
	if e == nil {
		return "yt-dlp command failed"
	}
	if e.Diagnostic == "" {
		return fmt.Sprintf("yt-dlp command failed: %v", e.Cause)
	}
	return fmt.Sprintf("yt-dlp command failed: %v: %s", e.Cause, e.Diagnostic)
}

// Unwrap returns the underlying process error so callers can classify missing
// executables, exit failures, and cancellation using the standard errors API.
func (e *CommandError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Run executes executable with args and returns its captured output. The
// context controls process cancellation and deadlines.
func (ExecRunner) Run(ctx context.Context, executable string, args ...string) (CommandResult, error) {
	command := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := CommandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err != nil {
		return result, &CommandError{Cause: err, Diagnostic: safeDiagnostic(stderr.String())}
	}
	return result, nil
}

// safeDiagnostic converts process stderr into a bounded single-line message
// while removing URLs and common credential-bearing header or option values.
func safeDiagnostic(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = secretDiagnosticPattern.ReplaceAllString(value, "$1$2[redacted]")
	value = urlDiagnosticPattern.ReplaceAllString(value, "[redacted-url]")
	if len(value) > maxDiagnosticLength {
		value = value[:maxDiagnosticLength] + "..."
	}
	return value
}
