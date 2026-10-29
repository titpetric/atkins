package runner

import (
	"context"
	"maps"
	"sync"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// jobTracker tracks job completion, thread-safe and shared across ExecutionContext copies.
type jobTracker struct {
	mu   sync.Mutex
	done map[string]bool
}

func newJobTracker() *jobTracker {
	return &jobTracker{done: make(map[string]bool)}
}

func (jt *jobTracker) Mark(jobName string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jt.done[jobName] = true
}

func (jt *jobTracker) IsCompleted(jobName string) bool {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	return jt.done[jobName]
}

// ExecutionContext holds the state of a run as a job and its steps see it.
//
// A job, a step and a loop iteration each run in a copy of their parent's
// context. Copy clones the variables and the environment, and a step's vars
// stay in the step. It shares the job tracker and the substitution memo, and
// every copy agrees on which jobs finished and on what a `$(...)`
// produced. A task name resolves in the skill's own pipeline first, then in
// every loaded pipeline; a leading colon skips the skill.
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

	// jobTracker tracks which jobs have finished execution (for dependency resolution).
	// Shared across copies so the mutex protects the map consistently.
	jobTracker *jobTracker

	// substitutions memoises the `$(...)` results of one step execution.
	// The passes that interpolate a step - the label, the condition,
	// the command - run the shell inside it once between them. Shared
	// across copies, and replaced for each step and each loop iteration.
	substitutions *substitutionCache

	// Progress receives job lifecycle events (optional).
	Progress ProgressObserver

	// Parents is the ancestor job chain for nested task invocations.
	Parents []string

	// Args are the command line arguments after a bare --. The exec:
	// steps of a job with args: true append them to their argv.
	Args []string
}

// Resolver resolves tasks in the execution context.
func (e *ExecutionContext) Resolver() *TaskResolver {
	return NewTaskResolver(e.AllPipelines)
}

// SkillResolver resolves tasks in skill context.
func (e *ExecutionContext) SkillResolver() *TaskResolver {
	return NewSkillResolver(e.Pipeline)
}

// Resolve returns the task a name resolves to, in the skill first, then globally.
func (e *ExecutionContext) Resolve(taskName string) (*model.ResolvedTask, error) {
	return e.SkillResolver().ResolveWithFallback(taskName, e.Resolver())
}

// Copy returns a copy one level deeper, without Context.
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
		jobTracker:   e.jobTracker,

		// Shared across clones: the copies a step makes of itself are the
		// passes that must agree on one answer.
		substitutions: e.substitutions,

		Progress: e.Progress,
		Parents:  append([]string(nil), e.Parents...),
		Args:     e.Args,
	}
}

// MarkJobCompleted marks a job as completed.
func (e *ExecutionContext) MarkJobCompleted(jobName string) {
	if e.jobTracker != nil {
		e.jobTracker.Mark(jobName)
	}
}

// IsJobCompleted checks if a job has completed.
func (e *ExecutionContext) IsJobCompleted(jobName string) bool {
	if e.jobTracker == nil {
		return false
	}
	return e.jobTracker.IsCompleted(jobName)
}

// Render refreshes the treeview.
func (e *ExecutionContext) Render() {
	e.Display.Render(e.Builder.Root())
}

// EmitProgress sends a job progress event if the context has an observer.
func (e *ExecutionContext) EmitProgress(ev JobProgressEvent) {
	if e.Progress != nil {
		e.Progress.OnJobProgress(ev)
	}
}

// NextStepIndex returns the next step index of the job, unique to each step and iteration.
func (e *ExecutionContext) NextStepIndex() int {
	e.stepSeqMu.Lock()
	defer e.stepSeqMu.Unlock()
	idx := e.StepSequence
	e.StepSequence++
	return idx
}
