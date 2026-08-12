package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const defaultExecutable = "yt-dlp"

var versionPattern = regexp.MustCompile(`^[0-9]{4}\.[0-9]{2}\.[0-9]{2}(?:[.+-][0-9A-Za-z.-]+)?$`)

// Client invokes a configured yt-dlp executable through an injectable runner.
// A Client is safe for concurrent use when its CommandRunner is safe for
// concurrent use.
type Client struct {
	executable string
	runner     CommandRunner
}

// NewClient constructs a client. An empty executable selects "yt-dlp" and a
// nil runner selects ExecRunner.
func NewClient(executable string, runner CommandRunner) *Client {
	if strings.TrimSpace(executable) == "" {
		executable = defaultExecutable
	}
	if runner == nil {
		runner = ExecRunner{}
	}
	return &Client{executable: executable, runner: runner}
}

// Version executes yt-dlp's machine-readable version command and validates
// that stdout contains a single date-based version identifier.
func (c *Client) Version(ctx context.Context) (string, error) {
	if c == nil || c.runner == nil {
		return "", errors.New("yt-dlp client is not configured")
	}
	result, err := c.runner.Run(ctx, c.executable, "--version")
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, exec.ErrDot) {
			return "", fmt.Errorf("yt-dlp executable %q was not found: %w", c.executable, err)
		}
		return "", fmt.Errorf("run yt-dlp --version: %w", err)
	}
	version := strings.TrimSpace(string(result.Stdout))
	if strings.ContainsAny(version, "\r\n") || !versionPattern.MatchString(version) {
		return "", fmt.Errorf("unexpected yt-dlp version output %q", version)
	}
	return version, nil
}
