package model

import "slices"

// JobStatus is the lifecycle state of a job. The values mirror the
// CHECK constraint in schema/job.up.sql; adding one means adding it in
// both places.
type JobStatus = string

// Job status values.
//
// A job starts pending. An agent claiming it moves it to running and
// holds a lease. From running it settles into exactly one terminal
// state: passed, failed, timeout or cancelled.
const (
	// JobStatusPending is a queued job that no agent has claimed.
	JobStatusPending JobStatus = "pending"

	// JobStatusRunning is a job claimed by an agent holding a lease.
	JobStatusRunning JobStatus = "running"

	// JobStatusPassed is a job whose command exited zero.
	JobStatusPassed JobStatus = "passed"

	// JobStatusFailed is a job whose command exited non-zero.
	JobStatusFailed JobStatus = "failed"

	// JobStatusTimeout is a job whose agent lease expired without a
	// terminal report. The server reclaims these for retry.
	JobStatusTimeout JobStatus = "timeout"

	// JobStatusCancelled is a job stopped before it settled.
	JobStatusCancelled JobStatus = "cancelled"
)

// jobStatuses is the full set, in lifecycle order.
var jobStatuses = []JobStatus{
	JobStatusPending,
	JobStatusRunning,
	JobStatusPassed,
	JobStatusFailed,
	JobStatusTimeout,
	JobStatusCancelled,
}

// terminalJobStatuses are the states a job settles into. A job in one
// of these is never picked up again; a retry creates a new job.
var terminalJobStatuses = []JobStatus{
	JobStatusPassed,
	JobStatusFailed,
	JobStatusTimeout,
	JobStatusCancelled,
}

// TerminalJobStatuses returns the states a job settles into. Retention
// sweeps use it: a job that has not settled is never old enough to
// delete, however long ago it was queued.
func TerminalJobStatuses() []JobStatus {
	statuses := make([]JobStatus, len(terminalJobStatuses))
	copy(statuses, terminalJobStatuses)
	return statuses
}

// ValidJobStatus reports whether status is a known job status.
func ValidJobStatus(status JobStatus) bool {
	return slices.Contains(jobStatuses, status)
}

// TerminalJobStatus reports whether status is a settled state.
func TerminalJobStatus(status JobStatus) bool {
	return slices.Contains(terminalJobStatuses, status)
}

// IsTerminal reports whether the job has settled.
func (j *Job) IsTerminal() bool {
	return TerminalJobStatus(j.Status)
}
