package transcript

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const whisperDiagnosticLimit = 2048

var whisperPathPattern = regexp.MustCompile(`(?i)([a-z]:\\|/)[^\s"']+`)

// WhisperCPPTranscriber transcribes audio with the whisper.cpp command-line
// program. It stages caller-owned audio in isolated temporary storage and uses
// FFmpeg to create 16 kHz mono PCM WAV input unless Audio.Format is "wav".
// The adapter is safe for concurrent use after construction.
type WhisperCPPTranscriber struct {
	executable string
	model      string
	ffmpeg     string
	runner     whisperCommandRunner
}

// NewWhisperCPPTranscriber constructs an opt-in local whisper.cpp adapter.
// executable names whisper-cli (or a compatible whisper.cpp executable),
// model is the required model file, and ffmpeg names the converter used for
// non-WAV input. Empty executable and ffmpeg values select "whisper-cli" and
// "ffmpeg", respectively. Model must not be empty.
func NewWhisperCPPTranscriber(executable, model, ffmpeg string) (*WhisperCPPTranscriber, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("%w: whisper.cpp model path must not be empty", ErrInvalidInput)
	}
	if strings.TrimSpace(executable) == "" {
		executable = "whisper-cli"
	}
	if strings.TrimSpace(ffmpeg) == "" {
		ffmpeg = "ffmpeg"
	}
	return &WhisperCPPTranscriber{executable: executable, model: model, ffmpeg: ffmpeg, runner: whisperExecRunner{}}, nil
}

// Transcribe stages audio, conditionally normalizes it with FFmpeg, invokes
// whisper.cpp for JSON segment output, and removes every staged artifact before
// returning. The first non-empty language hint is passed to whisper.cpp;
// otherwise language auto-detection is requested. Transcribe does not close
// audio.Data because ownership remains with the caller.
func (w *WhisperCPPTranscriber) Transcribe(ctx context.Context, audio Audio, languageHints []string) (Transcription, error) {
	if ctx == nil {
		return Transcription{}, fmt.Errorf("%w: context must not be nil", ErrInvalidInput)
	}
	if w == nil || w.runner == nil || strings.TrimSpace(w.executable) == "" || strings.TrimSpace(w.model) == "" {
		return Transcription{}, fmt.Errorf("%w: whisper.cpp transcriber is not configured", ErrInvalidInput)
	}
	if audio.Data == nil {
		return Transcription{}, fmt.Errorf("%w: audio data must not be nil", ErrInvalidInput)
	}
	directory, err := os.MkdirTemp("", "voxscripta-whisper-")
	if err != nil {
		return Transcription{}, fmt.Errorf("%w: create whisper.cpp workspace: %v", ErrProviderFailure, err)
	}
	defer os.RemoveAll(directory)

	format := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(audio.Format), "."))
	if format == "" {
		format = "audio"
	}
	inputPath := filepath.Join(directory, "input."+format)
	input, err := os.OpenFile(inputPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return Transcription{}, fmt.Errorf("%w: stage audio: %v", ErrProviderFailure, err)
	}
	_, copyErr := io.Copy(input, audio.Data)
	closeErr := input.Close()
	if copyErr != nil || closeErr != nil {
		return Transcription{}, fmt.Errorf("%w: stage audio: %v", ErrProviderFailure, errors.Join(copyErr, closeErr))
	}

	whisperInput := inputPath
	compatibleWAV, err := isWhisperCompatibleWAV(inputPath)
	if err != nil {
		return Transcription{}, fmt.Errorf("%w: inspect staged audio: %v", ErrProviderFailure, err)
	}
	if !compatibleWAV {
		whisperInput = filepath.Join(directory, "normalized.wav")
		if err := w.run(ctx, w.ffmpeg, "-nostdin", "-hide_banner", "-loglevel", "error", "-y", "-i", inputPath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", whisperInput); err != nil {
			return Transcription{}, classifyWhisperError("normalize audio with ffmpeg", err)
		}
	}

	outputPrefix := filepath.Join(directory, "transcript")
	language := firstLanguageHint(languageHints)
	args := []string{"--model", w.model, "--output-json", "--output-file", outputPrefix, "--language", language, "--file", whisperInput}
	if err := w.run(ctx, w.executable, args...); err != nil {
		return Transcription{}, classifyWhisperError("transcribe audio", err)
	}
	contents, err := os.ReadFile(outputPrefix + ".json")
	if err != nil {
		return Transcription{}, fmt.Errorf("%w: read whisper.cpp JSON output: %v", ErrProviderFailure, err)
	}
	return parseWhisperCPPJSON(contents, language)
}

// isWhisperCompatibleWAV reports whether path contains RIFF/WAVE audio with a
// PCM, mono, 16 kHz, 16-bit fmt chunk matching whisper.cpp's portable input
// contract. Unknown chunks are skipped without loading the complete file.
func isWhisperCompatibleWAV(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	header := make([]byte, 12)
	if _, err := io.ReadFull(file, header); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	if string(header[:4]) != "RIFF" || string(header[8:]) != "WAVE" {
		return false, nil
	}
	chunkHeader := make([]byte, 8)
	for {
		if _, err := io.ReadFull(file, chunkHeader); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
		size := int64(binary.LittleEndian.Uint32(chunkHeader[4:]))
		if string(chunkHeader[:4]) == "fmt " {
			if size < 16 {
				return false, nil
			}
			format := make([]byte, 16)
			if _, err := io.ReadFull(file, format); err != nil {
				return false, nil
			}
			return binary.LittleEndian.Uint16(format[0:]) == 1 &&
				binary.LittleEndian.Uint16(format[2:]) == 1 &&
				binary.LittleEndian.Uint32(format[4:]) == 16000 &&
				binary.LittleEndian.Uint16(format[14:]) == 16, nil
		}
		if size%2 != 0 {
			size++
		}
		if _, err := file.Seek(size, io.SeekCurrent); err != nil {
			return false, err
		}
	}
}

func (w *WhisperCPPTranscriber) run(ctx context.Context, executable string, args ...string) error {
	return w.runner.Run(ctx, executable, args...)
}

// firstLanguageHint returns the first non-blank ordered language preference or
// "auto" when the caller supplied no usable preference.
func firstLanguageHint(hints []string) string {
	for _, hint := range hints {
		if hint = strings.TrimSpace(hint); hint != "" {
			return hint
		}
	}
	return "auto"
}

type whisperJSON struct {
	Result struct {
		Language string `json:"language"`
	} `json:"result"`
	Transcription []struct {
		Offsets struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

// parseWhisperCPPJSON converts whisper.cpp's 10-millisecond offset units into
// normalized transcript segments and rejects output without usable speech.
func parseWhisperCPPJSON(contents []byte, requestedLanguage string) (Transcription, error) {
	var output whisperJSON
	if err := json.Unmarshal(contents, &output); err != nil {
		return Transcription{}, fmt.Errorf("%w: decode whisper.cpp JSON output: %v", ErrProviderFailure, err)
	}
	segments := make([]Segment, 0, len(output.Transcription))
	for _, item := range output.Transcription {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		segments = append(segments, Segment{Start: time.Duration(item.Offsets.From) * 10 * time.Millisecond, End: time.Duration(item.Offsets.To) * 10 * time.Millisecond, Text: text})
	}
	language := strings.TrimSpace(output.Result.Language)
	if language == "" && requestedLanguage != "auto" {
		language = requestedLanguage
	}
	if language == "" || language == "auto" || len(segments) == 0 {
		return Transcription{}, fmt.Errorf("%w: whisper.cpp returned no usable language or segments", ErrTranscriptUnavailable)
	}
	return Transcription{Language: Language{Code: language}, Provider: ProviderMetadata{Name: "whisper.cpp"}, Segments: segments}, nil
}

type whisperCommandRunner interface {
	Run(ctx context.Context, executable string, args ...string) error
}

type whisperExecRunner struct{}

// Run directly invokes one speech-processing command with bounded output and
// context-controlled cancellation.
func (whisperExecRunner) Run(ctx context.Context, executable string, args ...string) error {
	command := exec.CommandContext(ctx, executable, args...)
	stderr := &limitedWriter{limit: 64 << 10}
	command.Stdout = &limitedWriter{limit: 64 << 10}
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return &whisperCommandError{cause: err, diagnostic: safeWhisperDiagnostic(string(stderr.data))}
	}
	return nil
}

type limitedWriter struct {
	data  []byte
	limit int
}

// Write retains a bounded prefix while reporting the complete write consumed.
func (w *limitedWriter) Write(value []byte) (int, error) {
	length := len(value)
	remaining := w.limit - len(w.data)
	if remaining > len(value) {
		remaining = len(value)
	}
	if remaining > 0 {
		w.data = append(w.data, value[:remaining]...)
	}
	return length, nil
}

type whisperCommandError struct {
	cause      error
	diagnostic string
}

// Error renders a process failure with its optional sanitized diagnostic.
func (e *whisperCommandError) Error() string {
	if e.diagnostic == "" {
		return fmt.Sprintf("speech command failed: %v", e.cause)
	}
	return fmt.Sprintf("speech command failed: %v: %s", e.cause, e.diagnostic)
}

// Unwrap exposes the process cause for errors.Is and errors.As classification.
func (e *whisperCommandError) Unwrap() error { return e.cause }

// safeWhisperDiagnostic normalizes, path-redacts, and bounds process stderr.
func safeWhisperDiagnostic(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = whisperPathPattern.ReplaceAllString(value, "[redacted-path]")
	if len(value) > whisperDiagnosticLimit {
		value = value[:whisperDiagnosticLimit] + "..."
	}
	return value
}

// classifyWhisperError maps speech-process failures to stable public errors.
func classifyWhisperError(operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, exec.ErrDot) {
		return fmt.Errorf("%w: %s: %v", ErrMissingDependency, operation, err)
	}
	return fmt.Errorf("%w: %s: %v", ErrProviderFailure, operation, err)
}
