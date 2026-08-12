package ytdlp

import (
	"bytes"
	"context"
	"os/exec"
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

// Run executes executable with args and returns its captured output. The
// context controls process cancellation and deadlines.
func (ExecRunner) Run(ctx context.Context, executable string, args ...string) (CommandResult, error) {
	command := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return CommandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}
