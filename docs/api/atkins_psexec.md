# Package ./psexec

```go
import (
	"github.com/titpetric/atkins/psexec"
}
```

Package psexec provides process execution capabilities with support for
interactive terminals, PTY allocation, and streaming output over various
transports including websockets.

## Types

```go
// Command represents a command to be executed.
type Command struct {
	// Name is the command or executable name.
	Name	string
	// Args are the command arguments.
	Args	[]string
	// Dir is the working directory for the command.
	Dir	string
	// Env is the environment variables for the command.
	// Each entry should be in the form "KEY=VALUE".
	Env	[]string
	// Stdin is an optional reader for process input.
	Stdin	io.Reader
	// Stdout is an optional writer for stdout.
	// If nil, output is captured in Result.
	Stdout	io.Writer
	// Stderr is an optional writer for stderr.
	// If nil, output is captured in Result.
	Stderr	io.Writer
	// Timeout is the maximum duration for the command.
	// Zero means no timeout.
	Timeout	time.Duration
	// UsePTY enables pseudo-terminal allocation for the command.
	UsePTY	bool
	// Interactive enables full interactive mode with stdin/stdout binding.
	Interactive	bool
}
```

```go
// Executor manages process execution.
type Executor struct {
	// DefaultEnv is the default environment for all commands.
	DefaultEnv	[]string
	// DefaultDir is the default working directory for all commands.
	DefaultDir	string
}
```

```go
// Options configures the Executor.
type Options struct {
	// DefaultTimeout is the default timeout for commands.
	DefaultTimeout	time.Duration
	// DefaultDir is the default working directory for commands.
	DefaultDir	string
	// DefaultEnv is the default environment variables for commands.
	DefaultEnv	[]string
	// DefaultShell is the shell to use for shell commands.
	DefaultShell	string
}
```

```go
// Process represents a running process with PTY support.
type Process struct {
	cmd		*exec.Cmd
	ptmx		*os.File
	result		*processResult
	startTime	time.Time

	mu	sync.Mutex
	done	chan struct{}
	closed	bool
}
```

```go
// Result provides access to the outcome of a process execution.
type Result interface {
	// Output returns the combined stdout content.
	Output() string
	// ErrorOutput returns the stderr content.
	ErrorOutput() string
	// ExitCode returns the process exit code.
	ExitCode() int
	// Err returns any error that occurred during execution.
	Err() error
	// Success returns true if the process completed with exit code 0.
	Success() bool
	// Duration returns the execution duration.
	Duration() time.Duration
}
```

## Function symbols

- `func DefaultOptions () *Options`
- `func New () *Executor`
- `func NewCommand (name string, args ...string) *Command`
- `func NewShellCommand (script string) *Command`
- `func NewWithOptions (opts *Options) *Executor`
- `func (*Command) AsInteractive () *Command`
- `func (*Command) WithDir (dir string) *Command`
- `func (*Command) WithEnv (env []string) *Command`
- `func (*Command) WithPTY () *Command`
- `func (*Command) WithStderr (w io.Writer) *Command`
- `func (*Command) WithStdin (r io.Reader) *Command`
- `func (*Command) WithStdout (w io.Writer) *Command`
- `func (*Command) WithTimeout (d time.Duration) *Command`
- `func (*Executor) Run (ctx context.Context, cmd *Command) Result`
- `func (*Executor) RunWithIO (ctx context.Context, stdout io.Writer, stdin io.Reader, cmd *Command) Result`
- `func (*Executor) Start (ctx context.Context, cmd *Command) (*Process, error)`
- `func (*Process) Close () error`
- `func (*Process) Done () <-chan struct{}`
- `func (*Process) PID () int`
- `func (*Process) PTY () *os.File`
- `func (*Process) Pipe (stdout io.Writer, stdin io.Reader) error`
- `func (*Process) Read (b []byte) (int, error)`
- `func (*Process) Resize (rows,cols uint16) error`
- `func (*Process) Signal (sig os.Signal) error`
- `func (*Process) Wait () Result`
- `func (*Process) Write (b []byte) (int, error)`

### DefaultOptions

DefaultOptions returns the default options.

```go
func DefaultOptions () *Options
```

### New

New creates a new Executor with default settings.

```go
func New () *Executor
```

### NewCommand

NewCommand creates a new Command with the given name and arguments.

```go
func NewCommand (name string, args ...string) *Command
```

### NewShellCommand

NewShellCommand creates a new Command that runs via the shell.

```go
func NewShellCommand (script string) *Command
```

### NewWithOptions

NewWithOptions creates a new Executor with the given options.

```go
func NewWithOptions (opts *Options) *Executor
```

### AsInteractive

AsInteractive enables full interactive mode.

```go
func (*Command) AsInteractive () *Command
```

### WithDir

WithDir sets the working directory for the command.

```go
func (*Command) WithDir (dir string) *Command
```

### WithEnv

WithEnv sets the environment variables for the command.

```go
func (*Command) WithEnv (env []string) *Command
```

### WithPTY

WithPTY enables PTY allocation for the command.

```go
func (*Command) WithPTY () *Command
```

### WithStderr

WithStderr sets the stderr writer for the command.

```go
func (*Command) WithStderr (w io.Writer) *Command
```

### WithStdin

WithStdin sets the stdin reader for the command.

```go
func (*Command) WithStdin (r io.Reader) *Command
```

### WithStdout

WithStdout sets the stdout writer for the command.

```go
func (*Command) WithStdout (w io.Writer) *Command
```

### WithTimeout

WithTimeout sets the timeout for the command.

```go
func (*Command) WithTimeout (d time.Duration) *Command
```

### Run

Run executes a command and returns the result.

```go
func (*Executor) Run (ctx context.Context, cmd *Command) Result
```

### RunWithIO

RunWithIO executes a command with custom I/O streams, suitable for websocket transport.

```go
func (*Executor) RunWithIO (ctx context.Context, stdout io.Writer, stdin io.Reader, cmd *Command) Result
```

### Start

Start begins execution of a command and returns a Process handle.
The process can be used for bidirectional I/O, particularly useful
for websocket transport.

```go
func (*Executor) Start (ctx context.Context, cmd *Command) (*Process, error)
```

### Close

Close closes the PTY and terminates the process if still running.

```go
func (*Process) Close () error
```

### Done

Done returns a channel that is closed when the process completes.

```go
func (*Process) Done () <-chan struct{}
```

### PID

PID returns the process ID.

```go
func (*Process) PID () int
```

### PTY

PTY returns the PTY file handle for direct I/O.
This is useful for websocket transport where you want to
directly copy between the websocket and the PTY.

```go
func (*Process) PTY () *os.File
```

### Pipe

Pipe sets up bidirectional I/O between the process and the provided
reader/writer. This is the primary method for websocket integration.

```go
func (*Process) Pipe (stdout io.Writer, stdin io.Reader) error
```

### Read

Read reads from the process output (PTY).

```go
func (*Process) Read (b []byte) (int, error)
```

### Resize

Resize resizes the PTY window.

```go
func (*Process) Resize (rows,cols uint16) error
```

### Signal

Signal sends a signal to the process.

```go
func (*Process) Signal (sig os.Signal) error
```

### Wait

Wait waits for the process to complete and returns the result.

```go
func (*Process) Wait () Result
```

### Write

Write writes to the process input (PTY).

```go
func (*Process) Write (b []byte) (int, error)
```


