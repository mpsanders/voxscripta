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
	"time"

	transcript "github.com/mpsanders/VoxScripta"
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
		if err := encoder.Encode(result); err != nil {
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
