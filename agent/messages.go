package agent

import (
	"time"

	"github.com/titpetric/atkins/model"
)

// ExecutionStartMsg signals a task execution should begin.
type ExecutionStartMsg struct {
	Input    string // original user input
	Task     string
	Resolved *model.ResolvedTask
}

// ExecutionDoneMsg signals a task execution completed.
type ExecutionDoneMsg struct {
	Task     *model.ResolvedTask
	Err      error
	Duration time.Duration
}

// AutofixStartMsg signals an autofix should begin.
type AutofixStartMsg struct {
	OriginalTask *model.ResolvedTask
	FixTask      *model.ResolvedTask
}

// AutofixDoneMsg signals an autofix completed.
type AutofixDoneMsg struct {
	OriginalTask *model.ResolvedTask
	Err          error
	Duration     time.Duration
}

// RetryMsg signals a task should be retried.
type RetryMsg struct {
	Task *model.ResolvedTask
}

// ShellStartMsg signals a shell command should begin.
type ShellStartMsg struct {
	Command string
}

// ShellDoneMsg signals a shell command completed.
type ShellDoneMsg struct {
	Command  string
	Output   string
	Err      error
	ExitCode int
	Duration time.Duration
}
