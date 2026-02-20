package runner

import (
	"context"
	"sync"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ExecutionContext holds runtime state during pipeline Exec.
type ExecutionContext struct {
	Context context.Context

	Env     map[string]string
	Results map[string]any
	Verbose bool

	Variables map[string]any

	Pipeline *model.Pipeline
	Job      *model.Job
	Step     *model.Step

	Depth       int // Nesting depth for indentation
	StepsCount  int // Total number of steps executed
	StepsPassed int // Number of steps that passed

	CurrentJob  *treeview.TreeNode
	CurrentStep *treeview.Node

	Display     *treeview.Display
	Builder     *treeview.Builder
	JobNodes    map[string]*treeview.TreeNode // Map of job names to their tree nodes
	EventLogger *eventlog.Logger

	// Sequential step counter for this job (incremented for each step execution)
	StepSequence int
	stepSeqMu    sync.Mutex

	// JobCompleted tracks which jobs have finished execution (for dependency resolution)
	JobCompleted map[string]bool
	jobCompMu    sync.Mutex
}

// Copy copies everything except Context. Variables are shallow-copied.
// JobCompleted is shared (not copied) to maintain consistent dependency tracking.
func (e *ExecutionContext) Copy() *ExecutionContext {
	return &ExecutionContext{
		Variables:    copyVariables(e.Variables),
		Env:          copyEnv(e.Env),
		Results:      e.Results,
		Verbose:      e.Verbose,
		Pipeline:     e.Pipeline,
		Job:          e.Job,
		Step:         e.Step,
		Depth:        e.Depth + 1,
		StepsCount:   e.StepsCount,
		StepsPassed:  e.StepsPassed,
		CurrentJob:   e.CurrentJob,
		CurrentStep:  e.CurrentStep,
		Display:      e.Display,
		Builder:      e.Builder,
		JobNodes:     e.JobNodes,
		EventLogger:  e.EventLogger,
		StepSequence: e.StepSequence,
		JobCompleted: e.JobCompleted,
	}
}

// MarkJobCompleted marks a job as completed.
func (e *ExecutionContext) MarkJobCompleted(jobName string) {
	e.jobCompMu.Lock()
	defer e.jobCompMu.Unlock()
	if e.JobCompleted != nil {
		e.JobCompleted[jobName] = true
	}
}

// IsJobCompleted checks if a job has been completed.
func (e *ExecutionContext) IsJobCompleted(jobName string) bool {
	e.jobCompMu.Lock()
	defer e.jobCompMu.Unlock()
	if e.JobCompleted == nil {
		return false
	}
	return e.JobCompleted[jobName]
}

// Render refreshes the treeview.
func (e *ExecutionContext) Render() {
	e.Display.Render(e.Builder.Root())
}

// NextStepIndex returns the next sequential step index for this job execution.
// This ensures each step/iteration gets a unique number.
func (e *ExecutionContext) NextStepIndex() int {
	e.stepSeqMu.Lock()
	defer e.stepSeqMu.Unlock()
	idx := e.StepSequence
	e.StepSequence++
	return idx
}
