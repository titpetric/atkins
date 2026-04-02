package model

import pipeline "github.com/titpetric/atkins/model"

// ExecutionStartMsg signals a task execution should begin.
type ExecutionStartMsg struct {
	Input    string // original user input
	Task     string
	Resolved *pipeline.ResolvedTask
}
