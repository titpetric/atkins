package model

import pipeline "github.com/titpetric/atkins/model"

// RetryMsg signals a task should be retried.
type RetryMsg struct {
	Task *pipeline.ResolvedTask
}
