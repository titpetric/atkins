package model

import (
	"time"

	pipeline "github.com/titpetric/atkins/model"
)

// ExecutionDoneMsg signals a task execution completed.
type ExecutionDoneMsg struct {
	Task     *pipeline.ResolvedTask
	Err      error
	Duration time.Duration
}
