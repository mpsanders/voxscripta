package transcript

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseWhisperCPPJSON(t *testing.T) {
	tests := []struct {
		name      string
		contents  string
		requested string
		wantErr   error
		wantLang  string
		wantText  string
		wantStart time.Duration
		wantEnd   time.Duration
		wantCount int
	}{
		{name: "malformed", contents: "{", wantErr: ErrProviderFailure},
		{name: "empty", contents: `{}`, wantErr: ErrTranscriptUnavailable},
		{name: "auto without detected language", contents: `{"transcription":[{"offsets":{"from":0,"to":10},"text":"hello"}]}`, requested: "auto", wantErr: ErrTranscriptUnavailable},
		{name: "requested language fallback", contents: `{"transcription":[{"offsets":{"from":0,"to":10},"text":" hello "}]}`, requested: "en", wantLang: "en", wantText: "hello", wantEnd: 100 * time.Millisecond, wantCount: 1},
		{name: "detected language and offsets", contents: `{"result":{"language":"fr"},"transcription":[{"offsets":{"from":125,"to":250},"text":" bonjour "}]}`, requested: "en", wantLang: "fr", wantText: "bonjour", wantStart: 1250 * time.Millisecond, wantEnd: 2500 * time.Millisecond, wantCount: 1},
		{name: "empty segments skipped", contents: `{"result":{"language":"en"},"transcription":[{"offsets":{"from":0,"to":1},"text":" "},{"offsets":{"from":1,"to":2},"text":"ok"}]}`, wantLang: "en", wantText: "ok", wantStart: 10 * time.Millisecond, wantEnd: 20 * time.Millisecond, wantCount: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseWhisperCPPJSON([]byte(test.contents), test.requested)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("parseWhisperCPPJSON() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseWhisperCPPJSON() error = %v", err)
			}
			if got.Language.Code != test.wantLang || len(got.Segments) != test.wantCount {
				t.Fatalf("result = %#v, want language %q and %d segments", got, test.wantLang, test.wantCount)
			}
			segment := got.Segments[0]
			if segment.Text != test.wantText || segment.Start != test.wantStart || segment.End != test.wantEnd {
				t.Errorf("segment = %#v, want %q %s-%s", segment, test.wantText, test.wantStart, test.wantEnd)
			}
		})
	}
}

func TestWhisperCPPTranscriber(t *testing.T) {
	missing := errors.New("missing")
	tests := []struct {
		name          string
		format        string
		hints         []string
		nilContext    bool
		nilAudio      bool
		nilReceiver   bool
		failCommand   string
		failError     error
		writeJSON     string
		wantErr       error
		wantCommands  []string
		wantLanguage  string
		wantInputData []byte
		audioData     []byte
	}{
		{name: "nil context", nilContext: true, wantErr: ErrInvalidInput},
		{name: "nil audio", nilAudio: true, wantErr: ErrInvalidInput},
		{name: "nil receiver", nilReceiver: true, wantErr: ErrInvalidInput},
		{name: "compatible wav passthrough", format: "wav", audioData: compatibleWAV(), hints: []string{"en"}, writeJSON: whisperOutput("en", "hello"), wantCommands: []string{"whisper"}, wantLanguage: "en", wantInputData: compatibleWAV()},
		{name: "incompatible wav conversion", format: "wav", hints: []string{"en"}, writeJSON: whisperOutput("en", "hello"), wantCommands: []string{"ffmpeg", "whisper"}, wantLanguage: "en", wantInputData: []byte("normalized")},
		{name: "webm conversion", format: "webm", hints: []string{"", "de"}, writeJSON: whisperOutput("de", "hallo"), wantCommands: []string{"ffmpeg", "whisper"}, wantLanguage: "de", wantInputData: []byte("normalized")},
		{name: "ffmpeg missing", format: "m4a", failCommand: "ffmpeg", failError: &whisperCommandError{cause: exec.ErrNotFound}, wantErr: ErrMissingDependency, wantCommands: []string{"ffmpeg"}},
		{name: "whisper failure", format: "wav", audioData: compatibleWAV(), failCommand: "whisper", failError: missing, wantErr: ErrProviderFailure, wantCommands: []string{"whisper"}},
		{name: "invalid output", format: "wav", audioData: compatibleWAV(), writeJSON: `{`, wantErr: ErrProviderFailure, wantCommands: []string{"whisper"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeWhisperRunner{failCommand: test.failCommand, failError: test.failError, output: test.writeJSON}
			transcriber := &WhisperCPPTranscriber{executable: "whisper", model: "model.bin", ffmpeg: "ffmpeg", runner: runner}
			if test.nilReceiver {
				transcriber = nil
			}
			ctx := context.Background()
			if test.nilContext {
				ctx = nil
			}
			audioData := test.audioData
			if audioData == nil {
				audioData = []byte("audio")
			}
			audio := Audio{Data: io.NopCloser(bytes.NewReader(audioData)), Format: test.format}
			if test.nilAudio {
				audio.Data = nil
			}
			got, err := transcriber.Transcribe(ctx, audio, test.hints)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Transcribe() error = %v, want %v", err, test.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Transcribe() error = %v", err)
			}
			if got.Language.Code != test.wantLanguage {
				t.Errorf("language = %q, want %q", got.Language.Code, test.wantLanguage)
			}
			if !reflect.DeepEqual(runner.commands, test.wantCommands) {
				t.Errorf("commands = %q, want %q", runner.commands, test.wantCommands)
			}
			if test.wantInputData != nil && !bytes.Equal(runner.inputData, test.wantInputData) {
				t.Errorf("whisper input = %q, want %q", runner.inputData, test.wantInputData)
			}
			for _, directory := range runner.directories {
				if _, statErr := os.Stat(directory); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("workspace %q was not removed: %v", directory, statErr)
				}
			}
		})
	}
}

func TestNewWhisperCPPTranscriber(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		model      string
		ffmpeg     string
		wantErr    error
		wantExec   string
		wantFFmpeg string
	}{
		{name: "empty model", wantErr: ErrInvalidInput},
		{name: "whitespace model", model: " ", wantErr: ErrInvalidInput},
		{name: "defaults", model: "model.bin", wantExec: "whisper-cli", wantFFmpeg: "ffmpeg"},
		{name: "custom executables", executable: "whisper", model: "model.bin", ffmpeg: "avconv", wantExec: "whisper", wantFFmpeg: "avconv"},
		{name: "whitespace executable defaults", executable: " ", model: "model.bin", ffmpeg: " ", wantExec: "whisper-cli", wantFFmpeg: "ffmpeg"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewWhisperCPPTranscriber(test.executable, test.model, test.ffmpeg)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("NewWhisperCPPTranscriber() error = %v, want %v", err, test.wantErr)
			}
			if err == nil && (got.executable != test.wantExec || got.ffmpeg != test.wantFFmpeg) {
				t.Errorf("executables = %q/%q, want %q/%q", got.executable, got.ffmpeg, test.wantExec, test.wantFFmpeg)
			}
		})
	}
}

type fakeWhisperRunner struct {
	failCommand string
	failError   error
	output      string
	commands    []string
	directories []string
	inputData   []byte
}

func (r *fakeWhisperRunner) Run(_ context.Context, executable string, args ...string) error {
	r.commands = append(r.commands, executable)
	if executable == r.failCommand {
		return r.failError
	}
	if executable == "ffmpeg" {
		output := args[len(args)-1]
		r.directories = append(r.directories, filepath.Dir(output))
		return os.WriteFile(output, []byte("normalized"), 0o600)
	}
	var prefix, input string
	for index, arg := range args {
		if arg == "--output-file" {
			prefix = args[index+1]
		}
		if arg == "--file" {
			input = args[index+1]
		}
	}
	r.directories = append(r.directories, filepath.Dir(prefix))
	contents, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	r.inputData = contents
	return os.WriteFile(prefix+".json", []byte(r.output), 0o600)
}

func compatibleWAV() []byte {
	return []byte{'R', 'I', 'F', 'F', 36, 0, 0, 0, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' ', 16, 0, 0, 0, 1, 0, 1, 0, 0x80, 0x3e, 0, 0, 0, 0x7d, 0, 0, 2, 0, 16, 0, 'd', 'a', 't', 'a', 0, 0, 0, 0}
}

func whisperOutput(language, text string) string {
	return `{"result":{"language":"` + language + `"},"transcription":[{"offsets":{"from":0,"to":100},"text":"` + text + `"}]}`
}

func TestSafeWhisperDiagnostic(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{name: "empty"},
		{name: "whitespace", input: "one\n two", want: "one two"},
		{name: "unix path", input: "failed /tmp/private/input.wav", want: "failed [redacted-path]"},
		{name: "windows path", input: `failed C:\\private\\input.wav`, want: "failed [redacted-path]"},
		{name: "bounded", input: strings.Repeat("x", whisperDiagnosticLimit+10), want: strings.Repeat("x", whisperDiagnosticLimit) + "..."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := safeWhisperDiagnostic(test.input); got != test.want {
				t.Fatalf("safeWhisperDiagnostic() = %q, want %q", got, test.want)
			}
		})
	}
}
