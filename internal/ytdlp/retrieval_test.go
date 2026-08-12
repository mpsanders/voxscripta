package ytdlp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type retrievalRunner struct {
	err           error
	files         map[string]string
	args          []string
	outputDir     string
	outputIsAlive bool
}

// Run records the invocation and creates configured files beside the output
// template to emulate yt-dlp without executing a process.
func (r *retrievalRunner) Run(_ context.Context, _ string, args ...string) (CommandResult, error) {
	r.args = append([]string(nil), args...)
	for index, argument := range args {
		if argument != "--output" || index+1 >= len(args) {
			continue
		}
		r.outputDir = filepath.Dir(args[index+1])
		if _, err := os.Stat(r.outputDir); err == nil {
			r.outputIsAlive = true
		}
		for name, contents := range r.files {
			path := filepath.Join(r.outputDir, name)
			if strings.HasSuffix(name, string(os.PathSeparator)) {
				if err := os.Mkdir(path, 0o700); err != nil {
					return CommandResult{}, err
				}
				continue
			}
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				return CommandResult{}, err
			}
		}
	}
	return CommandResult{}, r.err
}

func TestClientRetrieve(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name       string
		client     *Client
		ctx        context.Context
		track      CaptionTrack
		files      map[string]string
		runErr     error
		want       string
		wantErr    bool
		wantCancel bool
		wantFlag   string
	}{
		{name: "manual caption", ctx: context.Background(), track: CaptionTrack{Language: "en-AU", Format: "vtt", Source: CaptionManual}, files: map[string]string{"caption.en-AU.vtt": "WEBVTT\n"}, want: "WEBVTT\n", wantFlag: "--write-subs"},
		{name: "automatic caption", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionAutomatic}, files: map[string]string{"caption.en.vtt": "automatic"}, want: "automatic", wantFlag: "--write-auto-subs"},
		{name: "missing output", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionManual}, wantErr: true},
		{name: "multiple outputs", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionManual}, files: map[string]string{"one.vtt": "one", "two.vtt": "two"}, wantErr: true},
		{name: "ignores unrelated output", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionManual}, files: map[string]string{"caption.en.vtt": "caption", "debug.txt": "debug", "nested" + string(os.PathSeparator): ""}, want: "caption", wantFlag: "--write-subs"},
		{name: "process failure", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionManual}, runErr: errors.New("exit status 1"), wantErr: true},
		{name: "cancellation", ctx: canceled, track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionManual}, runErr: errors.New("killed"), wantErr: true, wantCancel: true},
		{name: "empty language", ctx: context.Background(), track: CaptionTrack{Format: "vtt", Source: CaptionManual}, wantErr: true},
		{name: "unsupported format", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "json3", Source: CaptionManual}, wantErr: true},
		{name: "zero source", ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt"}, wantErr: true},
		{name: "nil client", client: nil, ctx: context.Background(), track: CaptionTrack{Language: "en", Format: "vtt", Source: CaptionManual}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &retrievalRunner{err: tt.runErr, files: tt.files}
			client := tt.client
			if tt.name != "nil client" {
				client = NewClient("yt-dlp", runner)
			}
			got, err := client.Retrieve(tt.ctx, "abcdefghijk", tt.track)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Retrieve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantCancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("Retrieve() error = %v, want context.Canceled", err)
			}
			if string(got) != tt.want {
				t.Errorf("Retrieve() = %q, want %q", got, tt.want)
			}
			if runner.outputDir != "" {
				if !runner.outputIsAlive {
					t.Error("temporary directory did not exist while runner executed")
				}
				if _, statErr := os.Stat(runner.outputDir); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("temporary directory cleanup error = %v, want os.ErrNotExist", statErr)
				}
			}
			if tt.wantFlag != "" && !containsArgument(runner.args, tt.wantFlag) {
				t.Errorf("runner args = %q, want %s", runner.args, tt.wantFlag)
			}
		})
	}
}

func TestSingleWebVTTPath(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    string
		wantErr bool
	}{
		{name: "one lowercase file", files: map[string]string{"caption.vtt": "x"}, want: "caption.vtt"},
		{name: "one uppercase file", files: map[string]string{"caption.VTT": "x"}, want: "caption.VTT"},
		{name: "ignores other extensions", files: map[string]string{"caption.vtt": "x", "notes.txt": "y"}, want: "caption.vtt"},
		{name: "no files", files: map[string]string{}, wantErr: true},
		{name: "only unrelated file", files: map[string]string{"notes.txt": "y"}, wantErr: true},
		{name: "multiple caption files", files: map[string]string{"one.vtt": "x", "two.vtt": "y"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			for name, contents := range tt.files {
				if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := singleWebVTTPath(directory)
			if (err != nil) != tt.wantErr {
				t.Fatalf("singleWebVTTPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want != "" && !reflect.DeepEqual(got, filepath.Join(directory, tt.want)) {
				t.Errorf("singleWebVTTPath() = %q, want %q", got, filepath.Join(directory, tt.want))
			}
		})
	}
}

// containsArgument reports whether arguments contains target.
func containsArgument(arguments []string, target string) bool {
	for _, argument := range arguments {
		if argument == target {
			return true
		}
	}
	return false
}
