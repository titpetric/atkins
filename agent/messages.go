package agent

import (
	"time"

	"github.com/titpetric/atkins/agent/ai"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
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

// JobProgressMsg signals a job progress update from the runner.
type JobProgressMsg struct {
	Event runner.JobProgressEvent
}

// JobProgressClosedMsg signals the progress channel was closed.
type JobProgressClosedMsg struct{}

// AIStartMsg signals the AI fallback should begin for an input local
// routing couldn't resolve.
type AIStartMsg struct {
	Input string
}

// AIDoneMsg signals claude answered. Result carries validated commands or a
// message; a claude failure, an unparseable reply and a command that does
// not invoke atkins all surface through Err.
type AIDoneMsg struct {
	Input    string
	Result   *ai.Result
	Err      error
	Duration time.Duration
}

// AICmdsDoneMsg signals the confirmed atkins commands finished running.
type AICmdsDoneMsg struct {
	Output   string
	Err      error
	Duration time.Duration
}
