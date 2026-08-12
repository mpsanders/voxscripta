package ytdlp

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

type stubRunner struct {
	result     CommandResult
	err        error
	executable string
	args       []string
}

// Run records the invocation and returns the configured result and error.
func (r *stubRunner) Run(_ context.Context, executable string, args ...string) (CommandResult, error) {
	r.executable = executable
	r.args = append([]string(nil), args...)
	return r.result, r.err
}

func TestClientVersion(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name       string
		client     *Client
		ctx        context.Context
		result     CommandResult
		runErr     error
		want       string
		wantErr    bool
		wantCancel bool
	}{
		{name: "stable version", ctx: context.Background(), result: CommandResult{Stdout: []byte("2026.08.12\n")}, want: "2026.08.12"},
		{name: "nightly version", ctx: context.Background(), result: CommandResult{Stdout: []byte("2026.08.12.232959")}, want: "2026.08.12.232959"},
		{name: "missing executable", ctx: context.Background(), runErr: exec.ErrNotFound, wantErr: true},
		{name: "process failure", ctx: context.Background(), runErr: errors.New("exit status 1"), wantErr: true},
		{name: "empty output", ctx: context.Background(), wantErr: true},
		{name: "multiple lines", ctx: context.Background(), result: CommandResult{Stdout: []byte("2026.08.12\nextra")}, wantErr: true},
		{name: "invalid version", ctx: context.Background(), result: CommandResult{Stdout: []byte("unknown")}, wantErr: true},
		{name: "canceled", ctx: canceled, runErr: errors.New("killed"), wantErr: true, wantCancel: true},
		{name: "nil client", client: nil, ctx: context.Background(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{result: tt.result, err: tt.runErr}
			client := tt.client
			if tt.name != "nil client" {
				client = NewClient("custom-yt-dlp", runner)
			}
			got, err := client.Version(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Version() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantCancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("Version() error = %v, want context.Canceled", err)
			}
			if got != tt.want {
				t.Errorf("Version() = %q, want %q", got, tt.want)
			}
			if tt.name != "nil client" && !reflect.DeepEqual(runner.args, []string{"--version"}) {
				t.Errorf("runner args = %q, want [--version]", runner.args)
			}
		})
	}
}

func TestNewClientDefaults(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		runner     CommandRunner
		wantPath   string
		wantExec   bool
	}{
		{name: "empty defaults", wantPath: defaultExecutable, wantExec: true},
		{name: "whitespace defaults", executable: "  ", wantPath: defaultExecutable, wantExec: true},
		{name: "custom path", executable: "tool", wantPath: "tool", wantExec: true},
		{name: "custom runner", runner: &stubRunner{}, wantPath: defaultExecutable},
		{name: "custom both", executable: "tool", runner: &stubRunner{}, wantPath: "tool"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient(tt.executable, tt.runner)
			if got.executable != tt.wantPath {
				t.Errorf("executable = %q, want %q", got.executable, tt.wantPath)
			}
			_, isExec := got.runner.(ExecRunner)
			if isExec != tt.wantExec {
				t.Errorf("runner is ExecRunner = %v, want %v", isExec, tt.wantExec)
			}
		})
	}
}
