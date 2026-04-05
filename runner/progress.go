package runner

import "time"

// JobProgressStatus represents the status of a job in progress.
type JobProgressStatus string

// Job progress status constants.
const (
	JobProgressRunning JobProgressStatus = "running"
	JobProgressPassed  JobProgressStatus = "passed"
	JobProgressFailed  JobProgressStatus = "failed"
	JobProgressSkipped JobProgressStatus = "skipped"
)

// JobProgressEvent represents a job lifecycle event.
type JobProgressEvent struct {
	JobName   string
	Parents   []string // ancestor chain for nested task invocations, e.g. ["default", "fmt"] when "lint" runs inside default > fmt
	Status    JobProgressStatus
	StartedAt time.Time
	Duration  time.Duration // set for terminal states
	Err       error         // set for failed
}

// ProgressObserver receives job progress events.
type ProgressObserver interface {
	OnJobProgress(JobProgressEvent)
}

// ProgressObserverFunc is a function adapter for ProgressObserver.
type ProgressObserverFunc func(JobProgressEvent)

// OnJobProgress implements ProgressObserver.
func (f ProgressObserverFunc) OnJobProgress(ev JobProgressEvent) { f(ev) }
