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
// EmptyResult is a Result for empty/no-op commands.
type EmptyResult struct{}
```

```go
// Executor manages process execution.
type Executor struct {
	// DefaultEnv is the default environment for all commands.
	DefaultEnv	[]string
	// DefaultDir is the default working directory for all commands.
	DefaultDir	string
	// DefaultTimeout is the default timeout for commands when not specified.
	// Zero means no timeout.
	DefaultTimeout	time.Duration
	// DefaultShell is the shell used for shell commands.
	// Defaults to "bash" if empty.
	DefaultShell	string
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
- `func (*Executor) Run (ctx context.Context, cmd *Command) Result`
- `func (*Executor) RunWithIO (ctx context.Context, stdout io.Writer, stdin io.Reader, cmd *Command) Result`
- `func (*Executor) ShellCommand (script string) *Command`
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
- `func (EmptyResult) Duration () time.Duration`
- `func (EmptyResult) Err () error`
- `func (EmptyResult) ErrorOutput () string`
- `func (EmptyResult) ExitCode () int`
- `func (EmptyResult) Output () string`
- `func (EmptyResult) Success () bool`

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

NewShellCommand creates a new Command that runs via bash.

```go
func NewShellCommand (script string) *Command
```

### NewWithOptions

NewWithOptions creates a new Executor with the given options.

```go
func NewWithOptions (opts *Options) *Executor
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

### ShellCommand

ShellCommand creates a new Command that runs via the executor's configured shell.

```go
func (*Executor) ShellCommand (script string) *Command
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

### Duration

Duration returns 0.

```go
func (EmptyResult) Duration () time.Duration
```

### Err

Err returns nil.

```go
func (EmptyResult) Err () error
```

### ErrorOutput

ErrorOutput returns empty string.

```go
func (EmptyResult) ErrorOutput () string
```

### ExitCode

ExitCode returns 0.

```go
func (EmptyResult) ExitCode () int
```

### Output

Output returns empty string.

```go
func (EmptyResult) Output () string
```

### Success

Success returns true.

```go
func (EmptyResult) Success () bool
```


