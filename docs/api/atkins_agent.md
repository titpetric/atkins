# Package ./agent

```go
import (
	"github.com/titpetric/atkins/agent"
}
```

## Types

<details>
<summary><code>type Agent</code></summary>

```go
// Agent manages the interactive REPL session.
type Agent struct {
	pipelines []*model.Pipeline
	resolver  *runner.TaskResolver
	options   *Options
	workDir   string
}
```

</details>

<details>
<summary><code>type AutoFixConfig</code></summary>

```go
// AutoFixConfig holds configuration for auto-fix behavior.
type AutoFixConfig struct {
	// Enabled controls whether auto-fix is attempted.
	Enabled bool

	// MaxRetries is the maximum number of fix+retry cycles.
	MaxRetries int

	// FixTaskSuffix is the suffix used to find fix tasks (default: "fix").
	FixTaskSuffix string
}
```

</details>

<details>
<summary><code>type AutoFixer</code></summary>

```go
// AutoFixer handles automatic error recovery.
type AutoFixer struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline
}
```

</details>

<details>
<summary><code>type AutofixDoneMsg</code></summary>

```go
// AutofixDoneMsg signals an autofix completed.
type AutofixDoneMsg struct {
	OriginalTask *model.ResolvedTask
	Err          error
	Duration     time.Duration
}
```

</details>

<details>
<summary><code>type AutofixStartMsg</code></summary>

```go
// AutofixStartMsg signals an autofix should begin.
type AutofixStartMsg struct {
	OriginalTask *model.ResolvedTask
	FixTask      *model.ResolvedTask
}
```

</details>

<details>
<summary><code>type ExecutionDoneMsg</code></summary>

```go
// ExecutionDoneMsg signals a task execution completed.
type ExecutionDoneMsg struct {
	Task     *model.ResolvedTask
	Err      error
	Duration time.Duration
}
```

</details>

<details>
<summary><code>type ExecutionProgressExported</code></summary>

```go
// ExecutionProgressExported is exported for testing. Use newExecutionProgress in production.
type ExecutionProgressExported = executionProgress
```

</details>

<details>
<summary><code>type ExecutionStartMsg</code></summary>

```go
// ExecutionStartMsg signals a task execution should begin.
type ExecutionStartMsg struct {
	Input    string // original user input
	Task     string
	Resolved *model.ResolvedTask
}
```

</details>

<details>
<summary><code>type Executor</code></summary>

```go
// Executor handles route execution with consistent output.
type Executor struct {
	agent   *Agent
	router  *router.Router
	out     Output
	workDir string
	ctx     context.Context
}
```

</details>

<details>
<summary><code>type GitStats</code></summary>

```go
// GitStats holds +/- line counts from git diff.
type GitStats struct {
	Added   int
	Removed int
}
```

</details>

<details>
<summary><code>type Greeter</code></summary>

```go
type (
	Greeter = greeting.Greeter
)
```

</details>

<details>
<summary><code>type JobProgressClosedMsg</code></summary>

```go
// JobProgressClosedMsg signals the progress channel was closed.
type JobProgressClosedMsg struct{}
```

</details>

<details>
<summary><code>type JobProgressMsg</code></summary>

```go
// JobProgressMsg signals a job progress update from the runner.
type JobProgressMsg struct {
	Event runner.JobProgressEvent
}
```

</details>

<details>
<summary><code>type Model</code></summary>

```go
// Model is the bubbletea model for the agent REPL.
type Model struct {
	agent      *Agent
	state      State
	input      string
	cursor     int
	history    []string
	historyIdx int
	breadcrumb *Breadcrumb
	lastError  error
	lastTask   *model.ResolvedTask
	retryCount int

	// Centralized router and slash commands
	router   *router.Router
	registry *Registry

	// Dimensions
	width  int
	height int

	// TUI state
	version   string
	hostname  string
	cwd       string
	gitBranch string
	gitStats  GitStats

	log             []LogEntry
	scrollOff       int
	spinner         spinner.Model
	progressSpinner spinner.Model
	runLogIdx       int // index of the current running entry in log

	// Confirmation state for fuzzy matching
	pendingConfirm *router.Route

	// Prompt mode (language or shell)
	promptMode PromptMode

	// Job progress tracking
	progressCh   <-chan runner.JobProgressEvent
	execProgress *executionProgress

	// Execution cancellation
	execCtx    context.Context
	execCancel context.CancelFunc

	// Double-cancel to quit tracking
	lastCancelTime time.Time
}
```

</details>

<details>
<summary><code>type Options</code></summary>

```go
// Options configures agent behavior.
type Options struct {
	Debug   bool
	Verbose bool
	Jail    bool
}
```

</details>

<details>
<summary><code>type Output</code></summary>

```go
// Output abstracts output operations for both interactive and non-interactive modes.
type Output interface {
	// Info prints informational output.
	Info(text string)
	// Error prints error output.
	Error(text string)
	// Prompt echoes the user's input.
	Prompt(text string)
	// CommandOutput prints command output (stdout/stderr).
	CommandOutput(text string)
}
```

</details>

<details>
<summary><code>type Registry</code></summary>

```go
// Registry is the slash command registry type.
type Registry = SlashRegistry
```

</details>

<details>
<summary><code>type RetryMsg</code></summary>

```go
// RetryMsg signals a task should be retried.
type RetryMsg struct {
	Task *model.ResolvedTask
}
```

</details>

<details>
<summary><code>type ShellDoneMsg</code></summary>

```go
// ShellDoneMsg signals a shell command completed.
type ShellDoneMsg struct {
	Command  string
	Output   string
	Err      error
	ExitCode int
	Duration time.Duration
}
```

</details>

<details>
<summary><code>type ShellResult</code></summary>

```go
// ShellResult contains the result of a shell command execution.
type ShellResult struct {
	Command  string
	Output   string
	ExitCode int
	Duration time.Duration
	Err      error
}
```

</details>

<details>
<summary><code>type ShellStartMsg</code></summary>

```go
// ShellStartMsg signals a shell command should begin.
type ShellStartMsg struct {
	Command string
}
```

</details>

<details>
<summary><code>type SlashCommand</code></summary>

```go
// SlashCommand represents a slash command handler.
type SlashCommand struct {
	Name        string
	Aliases     []string
	Description string
	Handler     func(m *Model, args string) (Model, tea.Cmd)
}
```

</details>

<details>
<summary><code>type SlashRegistry</code></summary>

```go
// SlashRegistry wraps the generic registry to implement CommandLookup.
type SlashRegistry struct {
	*registry.Registry[*SlashCommand]
}
```

</details>

<details>
<summary><code>type State</code></summary>

```go
// State represents the current REPL state.
type State int
```

</details>

<details>
<summary><code>type StdOutput</code></summary>

```go
// StdOutput implements Output for non-interactive mode (stdout/stderr).
type StdOutput struct {
	Out io.Writer
	Err io.Writer
}
```

</details>

<details>
<summary><code>type LogEntry, Breadcrumb, PromptMode, JobStatus, JobEntry, StepEntry, JobView</code></summary>

```go
// Type aliases for view package types.
type (
	LogEntry   = view.LogEntry
	Breadcrumb = view.Breadcrumb
	PromptMode = view.PromptMode
	JobStatus  = view.JobStatus
	JobEntry   = view.JobEntry
	StepEntry  = view.StepEntry
	JobView    = view.JobView
)
```

</details>

## Consts

<details>
<summary><code>const PromptModeLanguage, PromptModeShell</code></summary>

```go
// PromptMode constants.
const (
	PromptModeLanguage = view.PromptModeLanguage
	PromptModeShell    = view.PromptModeShell
)
```

</details>

<details>
<summary><code>const JobStatusPending, JobStatusRunning, JobStatusPassed, JobStatusFailed, JobStatusSkipped</code></summary>

```go
// JobStatus constants.
const (
	JobStatusPending = view.JobStatusPending
	JobStatusRunning = view.JobStatusRunning
	JobStatusPassed  = view.JobStatusPassed
	JobStatusFailed  = view.JobStatusFailed
	JobStatusSkipped = view.JobStatusSkipped
)
```

</details>

<details>
<summary><code>const StateIdle, StateExecuting, StateAutofix, StateRetrying</code></summary>

```go
// State constants for the REPL.
const (
	StateIdle State = iota
	StateExecuting
	StateAutofix
	StateRetrying
)
```

</details>

## Vars

<details>
<summary><code>var NewGreeter, MatchFortune, Fortune</code></summary>

```go
var (
	NewGreeter   = greeting.NewGreeter
	MatchFortune = greeting.MatchFortune
	Fortune      = greeting.Fortune
)
```

</details>

## Function symbols

- `func DefaultAutoFixConfig () *AutoFixConfig`
- `func DefaultRegistry () *Registry`
- `func DetectPromptMode (input string) PromptMode`
- `func New (workDir string, opts *Options) (*Agent, error)`
- `func NewAutoFixer (resolver *runner.TaskResolver, skills []*model.Pipeline) *AutoFixer`
- `func NewBreadcrumb () *Breadcrumb`
- `func NewExecutionProgressForTest () *executionProgress`
- `func NewExecutor (ctx context.Context, agent *Agent, rtr *router.Router, out Output) *Executor`
- `func NewJobView () *JobView`
- `func NewModel (agent *Agent, version string) Model`
- `func NewStdOutput () *StdOutput`
- `func RenderJobEntry (name string, running,failed bool, duration time.Duration, errMsg string) string`
- `func RenderJobSummary (total,passed,failed int, duration time.Duration) string`
- `func UsageText () string`
- `func (*Agent) Exec (ctx context.Context, prompt,version string) error`
- `func (*Agent) Options () *Options`
- `func (*Agent) Pipelines () []*model.Pipeline`
- `func (*Agent) Resolver () *runner.TaskResolver`
- `func (*Agent) Run (ctx context.Context, version string) error`
- `func (*Agent) WorkDir () string`
- `func (*AutoFixer) AttemptFix (ctx context.Context, task *model.ResolvedTask, allPipelines []*model.Pipeline) error`
- `func (*AutoFixer) CanFix (task *model.ResolvedTask) bool`
- `func (*AutoFixer) GetFixTask (task *model.ResolvedTask) (*model.ResolvedTask, error)`
- `func (*Executor) ExecuteRoute (route *router.Route) error`
- `func (*Executor) ExecuteShell (command string) error`
- `func (*Executor) ExecuteShellCapture (command string) ShellResult`
- `func (*Executor) ExecuteTask (task *model.ResolvedTask) error`
- `func (*Registry) HelpText () string`
- `func (*SlashRegistry) HasCommand (name string) bool`
- `func (*StdOutput) CommandOutput (text string)`
- `func (*StdOutput) Error (text string)`
- `func (*StdOutput) Info (text string)`
- `func (*StdOutput) Prompt (text string)`
- `func (Model) Init () tea.Cmd`
- `func (Model) Update (msg tea.Msg) (tea.Model, tea.Cmd)`
- `func (Model) View () tea.View`

### DefaultAutoFixConfig

DefaultAutoFixConfig returns the default auto-fix configuration.

```go
func DefaultAutoFixConfig() *AutoFixConfig
```

### DefaultRegistry

DefaultRegistry returns the built-in slash commands.

```go
func DefaultRegistry() *Registry
```

### DetectPromptMode

DetectPromptMode returns the appropriate mode based on input.

```go
func DetectPromptMode(input string) PromptMode
```

### New

New creates a new Agent with discovered skills.

```go
func New(workDir string, opts *Options) (*Agent, error)
```

### NewAutoFixer

NewAutoFixer creates a new auto-fixer.

```go
func NewAutoFixer(resolver *runner.TaskResolver, skills []*model.Pipeline) *AutoFixer
```

### NewBreadcrumb

NewBreadcrumb creates a new breadcrumb tracker.

```go
func NewBreadcrumb() *Breadcrumb
```

### NewExecutionProgressForTest

NewExecutionProgressForTest creates an executionProgress for testing.

```go
func NewExecutionProgressForTest() *executionProgress
```

### NewExecutor

NewExecutor creates a new executor.

```go
func NewExecutor(ctx context.Context, agent *Agent, rtr *router.Router, out Output) *Executor
```

### NewJobView

NewJobView creates a new job view.

```go
func NewJobView() *JobView
```

### NewModel

NewModel creates a new bubbletea model for the agent.

```go
func NewModel(agent *Agent, version string) Model
```

### NewStdOutput

NewStdOutput creates a new stdout-based output.

```go
func NewStdOutput() *StdOutput
```

### RenderJobEntry

RenderJobEntry renders a single job entry in gotestsum style.

```go
func RenderJobEntry(name string, running, failed bool, duration time.Duration, errMsg string) string
```

### RenderJobSummary

RenderJobSummary renders a summary line in gotestsum style.

```go
func RenderJobSummary(total, passed, failed int, duration time.Duration) string
```

### UsageText

UsageText returns the usage help text for non-interactive mode.

```go
func UsageText() string
```

### Exec

Exec processes a single prompt non-interactively and exits.

```go
func (*Agent) Exec(ctx context.Context, prompt, version string) error
```

### Options

Options returns the agent options.

```go
func (*Agent) Options() *Options
```

### Pipelines

Pipelines returns the loaded pipelines.

```go
func (*Agent) Pipelines() []*model.Pipeline
```

### Resolver

Resolver returns the task resolver.

```go
func (*Agent) Resolver() *runner.TaskResolver
```

### Run

Run starts the interactive REPL.

```go
func (*Agent) Run(ctx context.Context, version string) error
```

### WorkDir

WorkDir returns the working directory.

```go
func (*Agent) WorkDir() string
```

### AttemptFix

AttemptFix runs the fix task and reports success.

```go
func (*AutoFixer) AttemptFix(ctx context.Context, task *model.ResolvedTask, allPipelines []*model.Pipeline) error
```

### CanFix

CanFix checks if a fix skill exists for the given task.
e.g., if "go:test" fails, checks for "go:fix".

```go
func (*AutoFixer) CanFix(task *model.ResolvedTask) bool
```

### GetFixTask

GetFixTask returns the fix task for a given skill.
"go:test" -> "go:fix", "docker:build" -> "docker:fix".

```go
func (*AutoFixer) GetFixTask(task *model.ResolvedTask) (*model.ResolvedTask, error)
```

### ExecuteRoute

ExecuteRoute handles a routed command and returns an error if execution failed.

```go
func (*Executor) ExecuteRoute(route *router.Route) error
```

### ExecuteShell

ExecuteShell runs a shell command.

```go
func (*Executor) ExecuteShell(command string) error
```

### ExecuteShellCapture

ExecuteShellCapture runs a shell command and captures output.

```go
func (*Executor) ExecuteShellCapture(command string) ShellResult
```

### ExecuteTask

ExecuteTask runs a resolved task.

```go
func (*Executor) ExecuteTask(task *model.ResolvedTask) error
```

### HelpText

HelpText returns formatted help text for all commands.

```go
func (*Registry) HelpText() string
```

### HasCommand

HasCommand returns true if the command exists in the registry.

```go
func (*SlashRegistry) HasCommand(name string) bool
```

### Init

Init implements tea.Model.

```go
func (Model) Init() tea.Cmd
```

### Update

Update implements tea.Model.

```go
func (Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

### View

View implements tea.Model.

```go
func (Model) View() tea.View
```

### CommandOutput

```go
func (*StdOutput) CommandOutput(text string)
```

### Error

```go
func (*StdOutput) Error(text string)
```

### Info

```go
func (*StdOutput) Info(text string)
```

### Prompt

```go
func (*StdOutput) Prompt(text string)
```
