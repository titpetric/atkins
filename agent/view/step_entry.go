package view

import "time"

// StepEntry tracks a single step within a job.
type StepEntry struct {
	Name     string
	Status   JobStatus
	Duration time.Duration
	Error    string
}
