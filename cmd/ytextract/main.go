// Command ytextract is the development and diagnostic CLI for VoxScripta.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	transcript "github.com/mpsanders/VoxScripta"
	"github.com/mpsanders/VoxScripta/internal/ytdlp"
)

var version = "dev"

// main parses command-line arguments and delegates execution to run.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type languagesFlag []string

// String renders the configured language preferences for flag diagnostics.
func (languages languagesFlag) String() string {
	return fmt.Sprint([]string(languages))
}

// Set appends one language preference in command-line order.
func (languages *languagesFlag) Set(value string) error {
	if value == "" {
		return errors.New("language must not be empty")
	}
	*languages = append(*languages, value)
	return nil
}

// run executes the CLI with args, writes output to the supplied streams, and
// returns a process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ytextract", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print version")
	checkDependency := flags.Bool("check", false, "check yt-dlp availability and version")
	format := flags.String("format", "text", "output format: text or json")
	timeout := flags.Duration("timeout", 30*time.Second, "acquisition timeout")
	ytdlpPath := flags.String("yt-dlp", "yt-dlp", "yt-dlp executable path")
	manualOnly := flags.Bool("manual-only", false, "exclude automatic captions")
	var languages languagesFlag
	flags.Var(&languages, "language", "preferred caption language (repeatable)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if *checkDependency {
		if flags.NArg() != 0 || *timeout <= 0 {
			fmt.Fprintln(stderr, "usage: ytextract --check [--yt-dlp PATH] [--timeout DURATION]")
			return 2
		}
		return checkYTDLP(*ytdlpPath, *timeout, stdout, stderr)
	}
	if flags.NArg() != 1 || (*format != "text" && *format != "json") || *timeout <= 0 {
		fmt.Fprintln(stderr, "usage: ytextract [options] VIDEO_URL_OR_ID")
		return 2
	}
	client, err := transcript.New(transcript.WithYTDLPPath(*ytdlpPath))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCode(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := client.Get(ctx, flags.Arg(0), transcript.Options{
		Languages: languages, AllowAutomatic: !*manualOnly,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCode(err)
	}
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(newJSONTranscript(result)); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if _, err := fmt.Fprintln(stdout, result.Text()); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type jsonSegment struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Text  string `json:"text"`
}

type jsonTranscript struct {
	VideoID  string                      `json:"video_id"`
	Title    string                      `json:"title,omitempty"`
	Language transcript.Language         `json:"language"`
	Source   transcript.SourceKind       `json:"source"`
	Provider transcript.ProviderMetadata `json:"provider"`
	Segments []jsonSegment               `json:"segments"`
}

// newJSONTranscript builds the stable CLI JSON representation of result.
// Segment timestamps use readable duration strings instead of the domain
// model's integer nanosecond encoding.
func newJSONTranscript(result transcript.Transcript) jsonTranscript {
	segments := make([]jsonSegment, len(result.Segments))
	for index, segment := range result.Segments {
		segments[index] = jsonSegment{Start: segment.Start.String(), End: segment.End.String(), Text: segment.Text}
	}
	return jsonTranscript{
		VideoID: result.VideoID, Title: result.Title, Language: result.Language,
		Source: result.Source, Provider: result.Provider, Segments: segments,
	}
}

// checkYTDLP verifies that path names an executable yt-dlp installation and
// reports its version. Timeout bounds the subprocess probe; stdout and stderr
// receive the user-facing result and diagnostic, respectively.
func checkYTDLP(path string, timeout time.Duration, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	provider := ytdlp.NewClient(path, nil)
	providerVersion, err := provider.Version(ctx)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, exec.ErrDot) {
			err = fmt.Errorf("%w: yt-dlp executable %q: %v", transcript.ErrMissingDependency, path, err)
		}
		fmt.Fprintln(stderr, err)
		return exitCode(err)
	}
	if strings.TrimSpace(providerVersion) == "" {
		err = fmt.Errorf("%w: yt-dlp returned an empty version", transcript.ErrProviderFailure)
		fmt.Fprintln(stderr, err)
		return exitCode(err)
	}
	fmt.Fprintf(stdout, "yt-dlp %s ready\n", providerVersion)
	return 0
}

// exitCode maps public library errors to stable command exit codes.
func exitCode(err error) int {
	switch {
	case errors.Is(err, transcript.ErrInvalidInput):
		return 2
	case errors.Is(err, transcript.ErrMissingDependency):
		return 3
	case errors.Is(err, transcript.ErrTranscriptUnavailable):
		return 4
	default:
		return 1
	}
}
