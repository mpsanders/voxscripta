package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLISmoke builds the command and exercises its public process boundary
// against a deterministic fake yt-dlp executable.
func TestCLISmoke(t *testing.T) {
	commandPath, providerPath := buildSmokeBinaries(t)
	tests := []struct {
		name       string
		mode       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "text output", mode: "success", args: []string{"abcdefghijk"}, wantStdout: "Hello world\n"},
		{name: "dependency check", mode: "success", args: []string{"--check"}, wantStdout: "yt-dlp 2026.07.04 ready\n"},
		{name: "JSON output", mode: "success", args: []string{"--format", "json", "abcdefghijk"}, wantStdout: `"end": "1s"`},
		{name: "manual only", mode: "success", args: []string{"--manual-only", "abcdefghijk"}, wantStdout: "Hello world\n"},
		{name: "missing dependency", args: []string{"--yt-dlp", filepath.Join(t.TempDir(), "missing"), "abcdefghijk"}, wantCode: 3, wantStderr: "missing dependency"},
		{name: "unavailable transcript", mode: "unavailable", args: []string{"abcdefghijk"}, wantCode: 4, wantStderr: "transcript unavailable"},
		{name: "malformed metadata", mode: "malformed", args: []string{"abcdefghijk"}, wantCode: 1, wantStderr: "provider failure"},
		{name: "provider process failure", mode: "failure", args: []string{"abcdefghijk"}, wantCode: 1, wantStderr: "provider failure"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"--yt-dlp", providerPath}, test.args...)
			if test.name == "missing dependency" {
				args = test.args
			}
			command := exec.Command(commandPath, args...)
			command.Env = append(os.Environ(), "VOXSCRIPTA_FAKE_MODE="+test.mode)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			gotCode := 0
			if err != nil {
				var exitError *exec.ExitError
				if !strings.Contains(err.Error(), "exit status") || !asExitError(err, &exitError) {
					t.Fatalf("run CLI: %v", err)
				}
				gotCode = exitError.ExitCode()
			}
			if gotCode != test.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr = %q", gotCode, test.wantCode, stderr.String())
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

// asExitError reports whether err is an executable exit error.
func asExitError(err error, target **exec.ExitError) bool {
	exitError, ok := err.(*exec.ExitError)
	if ok {
		*target = exitError
	}
	return ok
}

// buildSmokeBinaries builds the CLI and its fake provider in a test-owned
// directory and returns their paths.
func buildSmokeBinaries(t *testing.T) (string, string) {
	t.Helper()
	directory := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	commandPath := filepath.Join(directory, "ytextract"+suffix)
	buildCommand := exec.Command("go", "build", "-o", commandPath, ".")
	if output, err := buildCommand.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v: %s", err, output)
	}

	helperDirectory := filepath.Join(directory, "fake-ytdlp")
	if err := os.Mkdir(helperDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	helperSource := `package main
import ("fmt"; "os"; "path/filepath"; "strings")
func main() {
 mode := os.Getenv("VOXSCRIPTA_FAKE_MODE")
 args := os.Args[1:]
 for _, arg := range args { if arg == "--version" { fmt.Println("2026.07.04"); return } }
 for _, arg := range args { if arg == "--dump-single-json" {
  if mode == "failure" { fmt.Fprintln(os.Stderr, "upstream failed"); os.Exit(1) }
  if mode == "malformed" { fmt.Print("{"); return }
  if mode == "unavailable" { fmt.Print("{\"id\":\"abcdefghijk\"}"); return }
  fmt.Print("{\"id\":\"abcdefghijk\",\"title\":\"Example\",\"language\":\"en\",\"subtitles\":{\"en\":[{\"ext\":\"vtt\",\"name\":\"English\",\"url\":\"https://example.invalid/caption\"}]}}"); return
 } }
 for i, arg := range args { if arg == "--output" && i+1 < len(args) {
  path := strings.Replace(args[i+1], "%(ext)s", "en.vtt", 1)
  if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { panic(err) }
  if err := os.WriteFile(path, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nHello world\n"), 0600); err != nil { panic(err) }
  return
 } }
 os.Exit(1)
}`
	sourcePath := filepath.Join(helperDirectory, "main.go")
	if err := os.WriteFile(sourcePath, []byte(helperSource), 0o600); err != nil {
		t.Fatal(err)
	}
	providerPath := filepath.Join(directory, "fake-yt-dlp"+suffix)
	buildProvider := exec.Command("go", "build", "-o", providerPath, ".")
	buildProvider.Dir = helperDirectory
	buildProvider.Env = append(os.Environ(), "GOWORK=off", "GO111MODULE=off")
	if output, err := buildProvider.CombinedOutput(); err != nil {
		t.Fatalf("build fake yt-dlp: %v: %s", err, output)
	}
	return commandPath, providerPath
}
