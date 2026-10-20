package psexec

import (
	"io"
	"time"
)

// Command describes a command for an Executor to run.
type Command struct {
	// Name is the command or executable name.
	Name string
	// Args are the command arguments.
	Args []string
	// Dir is the working directory for the command.
	Dir string
	// Env is the environment variables for the command.
	// Each entry takes the form "KEY=VALUE".
	Env []string
	// Stdin is an optional reader for process input.
	Stdin io.Reader
	// Stdout is an optional writer for stdout.
	// If nil, Result captures the output.
	Stdout io.Writer
	// Stderr is an optional writer for stderr.
	// If nil, Result captures the output.
	Stderr io.Writer
	// Timeout is the longest the command may run.
	// Zero means no timeout.
	Timeout time.Duration
	// UsePTY enables pseudo-terminal allocation for the command.
	UsePTY bool
	// Interactive enables full interactive mode with stdin/stdout binding.
	Interactive bool
}

// NewCommand creates a new Command with the given name and arguments.
func NewCommand(name string, args ...string) *Command {
	return &Command{
		Name: name,
		Args: args,
	}
}

// NewShellCommand creates a new Command that runs via bash.
func NewShellCommand(script string) *Command {
	return &Command{
		Name: "bash",
		Args: []string{"-c", script},
	}
}
