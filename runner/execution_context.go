package runner

import (
	"context"
	"maps"
	"sync"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// ExecutionContext holds runtime state during pipeline Exec.
type ExecutionContext struct {
	Context context.Context

	Env     Env
	Results map[string]any
	Verbose bool
	Dir     string

	Variables model.VariableStorage

	Pipeline     *model.Pipeline
	AllPipelines []*model.Pipeline // All loaded pipelines for cross-pipeline task references
	Job          *model.Job
	Step         *model.Step

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

// Resolver provides task resolution in the execution context.
func (e *ExecutionContext) Resolver() *TaskResolver {
	return NewTaskResolver(e.AllPipelines)
}

// SkillResolver provides task resolution in skill context.
func (e *ExecutionContext) SkillResolver() *TaskResolver {
	return NewSkillResolver(e.Pipeline)
}

// Resolve resolves a task name using skill-local scope first, then global.
// If the task has a ":" prefix, it resolves in global scope only (strict).
func (e *ExecutionContext) Resolve(taskName string) (*model.ResolvedTask, error) {
	return e.SkillResolver().ResolveWithFallback(taskName, e.Resolver())
}

// Copy copies everything except Context. Variables are cloned.
// JobCompleted is shared (not copied) to maintain consistent dependency tracking.
func (e *ExecutionContext) Copy() *ExecutionContext {
	var vars model.VariableStorage
	if e.Variables != nil {
		vars = e.Variables.Clone()
	}
	return &ExecutionContext{
		Variables:    vars,
		Env:          maps.Clone(e.Env),
		Results:      e.Results,
		Verbose:      e.Verbose,
		Dir:          e.Dir,
		Pipeline:     e.Pipeline,
		AllPipelines: e.AllPipelines,
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
