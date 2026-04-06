# Package ./agent/view

```go
import (
	"github.com/titpetric/atkins/agent/view"
}
```

## Types

```go
// Breadcrumb tracks execution progress as a one-liner.
type Breadcrumb struct {
	segments  []string
	status    string
	startTime time.Time
}
```

```go
// JobEntry tracks a single job's execution progress.
type JobEntry struct {
	Name      string
	Status    JobStatus
	StartTime time.Time
	Duration  time.Duration
	Error     string
	Steps     []StepEntry
}
```

```go
// JobStatus represents the execution status of a job.
type JobStatus int
```

```go
// JobView renders job execution in gotestsum-style format.
type JobView struct {
	entries []JobEntry
}
```

```go
// LogEntry represents a single entry in the message log.
type LogEntry struct {
	Kind     string // "info", "error", "run", "prompt", "output"
	Text     string
	Task     string
	Running  bool
	Duration time.Duration
	Failed   bool
	Progress string // live progress status line (when running)
}
```

```go
// PromptMode represents the current input mode.
type PromptMode int
```

```go
// RenderData holds all data needed to render the TUI view.
type RenderData struct {
	Width           int
	Height          int
	Version         string
	Hostname        string
	Cwd             string
	GitBranch       string
	GitAdded        int
	GitRemoved      int
	Log             []LogEntry
	ScrollOff       int
	Spinner         spinner.Model
	ProgressSpinner spinner.Model
	State           int // 0=idle
	Input           string
	Cursor          int
	PromptMode      PromptMode
}
```

```go
// StepEntry tracks a single step within a job.
type StepEntry struct {
	Name     string
	Status   JobStatus
	Duration time.Duration
	Error    string
}
```

## Consts

```go
// JobStatus constants.
const (
	JobStatusPending JobStatus = iota
	JobStatusRunning
	JobStatusPassed
	JobStatusFailed
	JobStatusSkipped
)
```

```go
// PromptMode constants.
const (
	// PromptModeLanguage is the default mode for natural language input.
	// Display character: >
	PromptModeLanguage PromptMode = iota

	// PromptModeShell is the mode for shell command input.
	// Display character: $ (deep orange)
	// Triggered when input starts with $.
	PromptModeShell
)
```

## Function symbols

- `func DetectPromptMode (input string) PromptMode`
- `func FormatDuration (d time.Duration) string`
- `func FormatJobDuration (d time.Duration) string`
- `func LogHeight (totalHeight int) int`
- `func NewBreadcrumb () *Breadcrumb`
- `func NewJobView () *JobView`
- `func Render (d *RenderData) tea.View`
- `func RenderFooter (promptMode PromptMode, w,gitAdded,gitRemoved,state,cursor int, cwd,gitBranch,input string) string`
- `func RenderJobEntry (name string, running bool, failed bool, duration time.Duration, errMsg string) string`
- `func RenderJobSummary (total,passed,failed int, duration time.Duration) string`
- `func RenderLog (spin spinner.Model, progressSpin spinner.Model, log []LogEntry, width int) []string`
- `func RenderRunEntry (spin spinner.Model, entry LogEntry) string`
- `func RenderWelcomeBox (text string, width int) []string`
- `func ShortenPath (p string) string`
- `func (*Breadcrumb) Clear ()`
- `func (*Breadcrumb) LastSegment () string`
- `func (*Breadcrumb) Pop ()`
- `func (*Breadcrumb) Push (segment string)`
- `func (*Breadcrumb) SetStatus (status string)`
- `func (*Breadcrumb) String () string`
- `func (*JobView) EndJob (name string, success bool, errMsg string)`
- `func (*JobView) StartJob (name string)`
- `func (PromptMode) PromptChar () string`

### DetectPromptMode

DetectPromptMode returns the appropriate mode based on input.
If input starts with "$", returns PromptModeShell.
Otherwise returns PromptModeLanguage.

```go
func DetectPromptMode(input string) PromptMode
```

### FormatDuration

FormatDuration formats a duration for display.

```go
func FormatDuration(d time.Duration) string
```

### FormatJobDuration

FormatJobDuration formats duration for display (similar to gotestsum).

```go
func FormatJobDuration(d time.Duration) string
```

### LogHeight

LogHeight calculates the available log area height.

```go
func LogHeight(totalHeight int) int
```

### NewBreadcrumb

NewBreadcrumb creates a new breadcrumb tracker.

```go
func NewBreadcrumb() *Breadcrumb
```

### NewJobView

NewJobView creates a new job view.

```go
func NewJobView() *JobView
```

### Render

Render produces the full TUI view.

```go
func Render(d *RenderData) tea.View
```

### RenderFooter

RenderFooter renders the 3-line footer (border + input + bottom border).

```go
func RenderFooter(promptMode PromptMode, w, gitAdded, gitRemoved, state, cursor int, cwd, gitBranch, input string) string
```

### RenderJobEntry

RenderJobEntry renders a single job entry in gotestsum style.

```go
func RenderJobEntry(name string, running bool, failed bool, duration time.Duration, errMsg string) string
```

### RenderJobSummary

RenderJobSummary renders a summary line in gotestsum style.

```go
func RenderJobSummary(total, passed, failed int, duration time.Duration) string
```

### RenderLog

RenderLog renders all log entries into lines.

```go
func RenderLog(spin spinner.Model, progressSpin spinner.Model, log []LogEntry, width int) []string
```

### RenderRunEntry

RenderRunEntry renders a single run log entry.

```go
func RenderRunEntry(spin spinner.Model, entry LogEntry) string
```

### RenderWelcomeBox

RenderWelcomeBox renders the welcome message in a bordered box.

```go
func RenderWelcomeBox(text string, width int) []string
```

### ShortenPath

ShortenPath replaces the home directory prefix with ~.

```go
func ShortenPath(p string) string
```

### Clear

Clear resets the breadcrumb.

```go
func (*Breadcrumb) Clear()
```

### LastSegment

LastSegment returns the last segment or empty string.

```go
func (*Breadcrumb) LastSegment() string
```

### Pop

Pop removes the last segment.

```go
func (*Breadcrumb) Pop()
```

### Push

Push adds a segment to the breadcrumb trail.
e.g., "go" -> "go > test" -> "go > test > step 1".

```go
func (*Breadcrumb) Push(segment string)
```

### SetStatus

SetStatus updates the current status suffix.
e.g., "running...", "passed", "failed".

```go
func (*Breadcrumb) SetStatus(status string)
```

### String

String renders the breadcrumb as a one-liner.
Output: "go > test > step 1 [running...]".

```go
func (*Breadcrumb) String() string
```

### EndJob

EndJob marks a job as complete.

```go
func (*JobView) EndJob(name string, success bool, errMsg string)
```

### StartJob

StartJob begins tracking a new job.

```go
func (*JobView) StartJob(name string)
```

### PromptChar

PromptChar returns the display character for the mode.

```go
func (PromptMode) PromptChar() string
```
