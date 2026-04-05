# Package ./agent/history

```go
import (
	"github.com/titpetric/atkins/agent/history"
}
```

## Types

```go
// ShellHistory maintains a persistent history of shell commands.
type ShellHistory struct {
	Entries []ShellHistoryEntry `json:"entries"`
	path    string
}
```

```go
// ShellHistoryEntry records a shell command execution.
type ShellHistoryEntry struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Dir      string        `json:"dir"`
	Time     time.Time     `json:"time"`
}
```

## Function symbols

- `func NewShellHistory () *ShellHistory`
- `func (*ShellHistory) Add (command string, exitCode int, duration time.Duration, dir string)`
- `func (*ShellHistory) FindExact (command string) *ShellHistoryEntry`
- `func (*ShellHistory) Match (input string) []ShellHistoryEntry`

### NewShellHistory

NewShellHistory loads or creates a shell history file.

```go
func NewShellHistory() *ShellHistory
```

### Add

Add records a command execution.

```go
func (*ShellHistory) Add(command string, exitCode int, duration time.Duration, dir string)
```

### FindExact

FindExact returns the most recent entry matching the exact command, or nil.

```go
func (*ShellHistory) FindExact(command string) *ShellHistoryEntry
```

### Match

Match returns shell history entries where the command starts with or
contains the given input. Results are returned most-recent-first.

```go
func (*ShellHistory) Match(input string) []ShellHistoryEntry
```
