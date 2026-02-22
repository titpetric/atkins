package psexec

import (
	"io"
	"time"
)

// Command represents a command to be executed.
type Command struct {
	// Name is the command or executable name.
	Name string
	// Args are the command arguments.
	Args []string
	// Dir is the working directory for the command.
	Dir string
	// Env is the environment variables for the command.
	// Each entry should be in the form "KEY=VALUE".
	Env []string
	// Stdin is an optional reader for process input.
	Stdin io.Reader
	// Stdout is an optional writer for stdout.
	// If nil, output is captured in Result.
	Stdout io.Writer
	// Stderr is an optional writer for stderr.
	// If nil, output is captured in Result.
	Stderr io.Writer
	// Timeout is the maximum duration for the command.
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

// NewShellCommand creates a new Command that runs via the shell.
func NewShellCommand(script string) *Command {
	return &Command{
		Name: "bash",
		Args: []string{"-c", script},
	}
}

// WithDir sets the working directory for the command.
func (c *Command) WithDir(dir string) *Command {
	c.Dir = dir
	return c
}

// WithEnv sets the environment variables for the command.
func (c *Command) WithEnv(env []string) *Command {
	c.Env = env
	return c
}

// WithStdin sets the stdin reader for the command.
func (c *Command) WithStdin(r io.Reader) *Command {
	c.Stdin = r
	return c
}

// WithStdout sets the stdout writer for the command.
func (c *Command) WithStdout(w io.Writer) *Command {
	c.Stdout = w
	return c
}

// WithStderr sets the stderr writer for the command.
func (c *Command) WithStderr(w io.Writer) *Command {
	c.Stderr = w
	return c
}

// WithTimeout sets the timeout for the command.
func (c *Command) WithTimeout(d time.Duration) *Command {
	c.Timeout = d
	return c
}

// WithPTY enables PTY allocation for the command.
func (c *Command) WithPTY() *Command {
	c.UsePTY = true
	return c
}

// AsInteractive enables full interactive mode.
func (c *Command) AsInteractive() *Command {
	c.Interactive = true
	c.UsePTY = true
	return c
}
