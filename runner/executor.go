package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/treeview"
)

// Executor runs pipeline jobs and steps.
type Executor struct {
	opts *Options
}

// NewExecutor creates a new executor with default options.
func NewExecutor() *Executor {
	return &Executor{
		opts: DefaultOptions(),
	}
}

// NewExecutorWithOptions creates a new executor with custom options.
func NewExecutorWithOptions(opts *Options) *Executor {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &Executor{
		opts: opts,
	}
}

// executeCommands executes a list of commands, updating child nodes if available.
// Returns the last error encountered (continues on errors to collect all failures).
func (e *Executor) executeCommands(ctx context.Context, stepCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, commands []string, stepIndex int) error {
	if len(commands) == 0 {
		return nil
	}

	// If step has multiple commands, update child nodes individually
	var cmdNodes []*treeview.Node
	if stepNode != nil {
		cmdNodes = stepNode.GetChildren()
	}

	var lastErr error
	for i, cmd := range commands {
		var cmdNode *treeview.Node
		if i < len(cmdNodes) {
			cmdNode = cmdNodes[i]
		} else if stepNode != nil {
			cmdNode = stepNode // Fallback to parent if no child nodes
		}
		if err := e.executeStepIteration(ctx, stepCtx, step, cmdNode, cmd, stepIndex+i); err != nil {
			lastErr = err
		}
	}

	// Update parent node status if we used child nodes
	if len(cmdNodes) > 0 && stepNode != nil {
		if lastErr != nil {
			stepNode.SetStatus(treeview.StatusFailed)
		} else {
			stepNode.SetStatus(treeview.StatusPassed)
		}
	}

	return lastErr
}

// parseTimeout parses a timeout string into a duration, using default if empty
func parseTimeout(timeoutStr string, defaultTimeout time.Duration) time.Duration {
	if timeoutStr == "" {
		return defaultTimeout
	}
	duration, err := time.ParseDuration(timeoutStr)
	if err != nil {
		// If parsing fails, return default
		return defaultTimeout
	}
	return duration
}

// ExecuteJob runs a single job.
func (e *Executor) ExecuteJob(parentCtx context.Context, execCtx *ExecutionContext) error {
	if execCtx == nil {
		return fmt.Errorf("execution context is nil")
	}

	job := execCtx.Job
	if job == nil {
		return fmt.Errorf("job is nil in execution context")
	}

	// Parse job timeout
	jobTimeout := parseTimeout(job.Timeout, e.opts.DefaultTimeout)

	// Create a child context with the job timeout
	ctx, cancel := context.WithTimeout(parentCtx, jobTimeout)
	defer cancel()

	// Store context in execution context for use in steps
	execCtx.Context = ctx

	// Evaluate job-level working directory and merge variables.
	// The order depends on whether dir references variables:
	// - Static dir (e.g., "/path"): evaluate dir first, then vars use that cwd
	// - Dynamic dir (e.g., "${{workdir}}"): evaluate vars first, then interpolate dir
	// When the job has a for loop, skip dir entirely — it may reference
	// loop variables (e.g., ${{folder}}) and will be evaluated per iteration.
	if job.For.IsEmpty() {
		if err := evaluateDirAndVars(execCtx, job, true); err != nil {
			return err
		}
	} else {
		// Still merge vars/env, but skip dir (deferred to per-iteration)
		savedDir := job.Dir
		job.Dir = ""
		if err := evaluateDirAndVars(execCtx, job, false); err != nil {
			job.Dir = savedDir
			return err
		}
		job.Dir = savedDir
	}

	// Execute steps - with optional job-level for loop.
	// When the job has a for loop, defer if/dir evaluation to each iteration
	// since they may reference loop variables (e.g., ${{folder}}).
	steps := job.Children()

	if !job.For.IsEmpty() {
		return e.executeJobWithForLoop(ctx, execCtx, steps)
	}

	// Evaluate job-level if condition
	shouldRun, err := EvaluateJobIf(execCtx)
	if err != nil {
		return fmt.Errorf("failed to evaluate if condition for job %q: %w", job.Name, err)
	}
	if !shouldRun {
		return ErrJobSkipped
	}

	return e.executeSteps(ctx, execCtx, steps)
}

// executeJobWithForLoop runs all job steps repeatedly for each iteration of the job-level for loop.
func (e *Executor) executeJobWithForLoop(ctx context.Context, execCtx *ExecutionContext, steps []*model.Step) error {
	job := execCtx.Job
	jobNode := execCtx.CurrentJob

	// Create a synthetic step to carry the job's For iterators for ExpandFor
	syntheticStep := &model.Step{
		For: job.For,
	}
	forCtx := execCtx.Copy()
	forCtx.Step = syntheticStep

	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: execCtx.Dir,
		DefaultEnv: execCtx.Env.Environ(),
	})
	iterations, err := ExpandFor(forCtx, func(script string) (string, error) {
		result := exec.Run(ctx, exec.ShellCommand(script))
		if !result.Success() {
			return "", NewExecError(result)
		}
		return result.Output(), nil
	})
	if err != nil {
		return fmt.Errorf("failed to expand job-level for loop for job %q: %w", job.Name, err)
	}

	if len(iterations) == 0 {
		return nil
	}

	// Replace pre-built step children with iteration sub-nodes
	jobNode.Node.ClearChildren()

	for idx, iteration := range iterations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		iterCtx := e.prepareIterationContext(execCtx, iteration.Variables)
		iterCtx.Context = ctx
		iterCtx.StepSequence = 0

		// Evaluate job-level if condition per iteration — it may reference loop variables
		if !job.If.IsEmpty() {
			shouldRun, err := EvaluateJobIf(iterCtx)
			if err != nil {
				return fmt.Errorf("failed to evaluate if condition for job %q: %w", job.Name, err)
			}
			if !shouldRun {
				continue
			}
		}

		// Re-evaluate job dir per iteration — it may reference loop variables
		if job.Dir != "" {
			dir, err := InterpolateString(job.Dir, iterCtx)
			if err != nil {
				return fmt.Errorf("failed to interpolate job dir %q for iteration: %w", job.Dir, err)
			}
			if err := validateDir(dir); err != nil {
				return fmt.Errorf("job dir %q: %w", dir, err)
			}
			iterCtx.Dir = dir
		}

		// Build iteration label from interpolated desc or job name
		iterLabel := fmt.Sprintf("iteration %d", idx)
		if job.Desc != "" {
			if interpolated, err := InterpolateString(job.Desc, iterCtx); err == nil {
				iterLabel = interpolated
			}
		}

		// Create iteration sub-node with its own step children
		iterNode := createIterationNode(
			fmt.Sprintf("jobs.%s.iter.%d", job.Name, idx),
			iterLabel,
			job.Summarize,
		)
		iterNode.SetStatus(treeview.StatusRunning)
		buildAndAddStepsToJob(&treeview.TreeNode{Node: iterNode}, steps)
		jobNode.AddChild(iterNode)

		// Point the iteration context at this sub-node so executeSteps finds step nodes
		iterCtx.CurrentJob = &treeview.TreeNode{Node: iterNode}
		execCtx.Render()

		if err := e.executeSteps(ctx, iterCtx, steps); err != nil {
			iterNode.SetStatus(treeview.StatusFailed)
			return err
		}
		iterNode.SetStatus(treeview.StatusPassed)
	}

	return nil
}

// evaluateDirAndVars uses lazy evaluation for job/task vars and dir.
// Vars are set as pending, dir is interpolated (resolving needed vars on-demand),
// then remaining vars are resolved for step execution.
// The label (e.g. "job", "task") is used in error messages.
// When checkDir is true, the resolved directory is validated to exist.
func evaluateDirAndVars(ctx *ExecutionContext, job *model.Job, checkDir bool, label ...string) error {
	prefix := "job"
	if len(label) > 0 {
		prefix = label[0]
	}
	// Set up lazy evaluation for vars
	if job.Decl != nil && job.Decl.Vars != nil {
		lazyVars := NewContextVariablesWithResolver(job.Decl.Vars, func(s string) (string, error) {
			return InterpolateString(s, ctx)
		})
		// Copy existing variables into the lazy storage
		ctx.Variables.Walk(func(k string, v any) {
			lazyVars.Set(k, v)
		})
		ctx.Variables = lazyVars
	}

	// Evaluate dir - this will lazily resolve any vars it references via Get
	if job.Dir != "" {
		dir, err := InterpolateString(job.Dir, ctx)
		if err != nil {
			return fmt.Errorf("failed to interpolate %s dir %q: %w", prefix, job.Dir, err)
		}
		if checkDir {
			if err := validateDir(dir); err != nil {
				return fmt.Errorf("%s dir %q: %w", prefix, dir, err)
			}
		}
		ctx.Dir = dir
	}

	// Process env vars (these need eager evaluation for shell access)
	if job.Decl != nil && job.Decl.Env != nil {
		if err := mergeEnv(ctx, job.Decl.Env); err != nil {
			return fmt.Errorf("error processing environment: %w", err)
		}
	}

	// Resolve any remaining pending vars now (ensures all vars are available for steps)
	if cv, ok := ctx.Variables.(*ContextVariables); ok {
		if err := cv.ResolveAll(); err != nil {
			return fmt.Errorf("failed to resolve variables: %w", err)
		}
	}
	return nil
}

// validateDir checks that a directory exists and is a directory.
func validateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

// executeSteps runs a sequence of steps (deferred steps are already at the end of the list)
func (e *Executor) executeSteps(ctx context.Context, execCtx *ExecutionContext, steps []*model.Step) error {
	eg := new(errgroup.Group)

	detached := 0
	deferredSteps := []*model.Step{}
	deferredIndices := []int{}

	// Wait for all detached steps to complete before running deferred steps.
	wait := func() error {
		if detached > 0 {
			if err := eg.Wait(); err != nil {
				return err
			}
			detached = 0
		}
		return nil
	}

	// First pass: execute non-detached steps and collect deferred steps
	for idx, step := range steps {
		if step.IsDeferred() {
			// Collect deferred steps for later execution
			deferredSteps = append(deferredSteps, step)
			deferredIndices = append(deferredIndices, idx)
			continue
		}

		if step.Detach {
			detached++
			step := steps[idx]
			idx := idx
			eg.Go(func() error {
				// Each detached task tree gets its own cancellable context
				// so that if an error occurs, only this tree is cancelled
				treeCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				return e.executeStep(treeCtx, execCtx, step, idx)
			})
			continue
		}

		if err := wait(); err != nil {
			return err
		}

		if err := e.executeStep(ctx, execCtx, steps[idx], idx); err != nil {
			return err
		}
	}

	if err := wait(); err != nil {
		return err
	}

	// Second pass: execute deferred steps after all detached steps are done
	for i, step := range deferredSteps {
		stepIdx := deferredIndices[i]

		// Find the deferred step node by looking for deferred nodes in the tree
		// We need to find it by matching deferred status, not by index (since for loops may have expanded)
		var stepNode *treeview.Node
		if execCtx.CurrentJob != nil {
			children := execCtx.CurrentJob.GetChildren()
			deferredCount := 0
			// Count deferred nodes to find the i-th deferred node
			for _, child := range children {
				if child.Deferred {
					if deferredCount == i {
						stepNode = child.Node
						break
					}
					deferredCount++
				}
			}
		}

		// Update status to running and re-render to show the transition
		stepNode.SetStatus(treeview.StatusRunning)

		// Execute step with the actual found node
		if err := e.executeStepWithNode(ctx, execCtx, step, stepNode); err != nil {
			return err
		} else {
			// Fallback to executeStep if node not found
			if err := e.executeStep(ctx, execCtx, step, stepIdx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeStepWithNode runs a single step with a provided node
func (e *Executor) executeStepWithNode(ctx context.Context, execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node) error {
	stepCtx, err := e.prepareStepContext(execCtx, ctx, step)
	if err != nil {
		stepNode.SetStatus(treeview.StatusFailed)
		return err
	}

	// Merge step-level vars with interpolation - but skip if step has a for loop
	// When !step.For.IsEmpty(), vars may depend on loop variables (e.g., ${{item}})
	// and should be merged inside the iteration context instead
	if step.For.IsEmpty() {
		if err := MergeVariables(stepCtx, step.Decl); err != nil {
			stepNode.SetStatus(treeview.StatusFailed)
			return fmt.Errorf("failed to process step env: %w", err)
		}
	}

	stepCtx.CurrentStep = stepNode

	// Evaluate if condition
	shouldRun, err := EvaluateIf(stepCtx)
	if err != nil {
		// If condition evaluation fails, skip the step
		stepNode.SetStatus(treeview.StatusSkipped)
		return fmt.Errorf("failed to evaluate if condition for step %q: %w", step.Name, err)
	}

	if !shouldRun {
		seqIndex := execCtx.NextStepIndex()
		e.logStepSkipped(execCtx, step, stepNode, seqIndex)
		return nil
	}

	// Handle for loop expansion
	if !step.For.IsEmpty() {
		return e.executeStepWithForLoop(ctx, stepCtx, step, stepNode, 0)
	} else {
		// Handle task invocation
		if step.Task != "" {
			stepNode.SetStatus(treeview.StatusRunning)
			return e.executeTaskStep(ctx, stepCtx, step, stepNode)
		}
	}

	// Execute all commands
	return e.executeCommands(ctx, stepCtx, step, stepNode, step.Commands(), 0)
}

// executeStep runs a single step
func (e *Executor) executeStep(ctx context.Context, execCtx *ExecutionContext, step *model.Step, stepIndex int) error {
	defer execCtx.Render()

	// Get the next sequential step index from the PARENT context before copying
	// This ensures all steps in a job get unique sequential indices
	seqIndex := execCtx.NextStepIndex()

	stepCtx, err := e.prepareStepContext(execCtx, ctx, step)
	if err != nil {
		return err
	}
	stepCtx.StepSequence = seqIndex // Set the index for this step

	// Get step node from tree, or create one on-demand for dynamically expanded iterations
	var stepNode *treeview.Node
	if jobNode := execCtx.CurrentJob; jobNode != nil {
		children := jobNode.GetChildren()
		if stepIndex < len(children) {
			stepNode = children[stepIndex].Node
		} else {
			stepNode = treeview.NewPendingStepNode(step.DisplayLabel(), step.IsDeferred(), step.Summarize)
			stepNode.SetQuiet(step.Quiet)
			jobNode.AddChild(stepNode)
		}
		stepCtx.CurrentStep = stepNode
	}

	// Merge step-level vars with interpolation - but skip if step has a for loop
	// When !step.For.IsEmpty(), vars may depend on loop variables (e.g., ${{item}})
	// and should be merged inside the iteration context instead
	if step.For.IsEmpty() {
		if err := MergeVariables(stepCtx, step.Decl); err != nil {
			stepNode.SetStatus(treeview.StatusFailed)
			return fmt.Errorf("failed to process step env: %w", err)
		}
	}

	// Interpolate step display label with current context
	if label := step.DisplayLabel(); label != "" {
		if interpolated, err := InterpolateCommand(label, stepCtx); err == nil {
			stepNode.SetName(interpolated)
		}
	}

	// Evaluate if condition
	shouldRun, err := EvaluateIf(stepCtx)
	if err != nil {
		// If condition evaluation fails, skip the step
		stepNode.SetStatus(treeview.StatusSkipped)
		return fmt.Errorf("failed to evaluate if condition for step %q: %w", step.Name, err)
	}

	if !shouldRun {
		e.logStepSkipped(execCtx, step, stepNode, seqIndex)
		return nil
	}

	// Handle task invocation
	if step.Task != "" {
		stepNode.SetStatus(treeview.StatusRunning)
		return e.executeTaskStep(ctx, stepCtx, step, stepNode)
	}

	// Handle for loop expansion
	if !step.For.IsEmpty() {
		stepNode.SetSummarize(step.Summarize)
		stepNode.SetStatus(treeview.StatusRunning)
		if err := e.executeStepWithForLoop(ctx, stepCtx, step, stepNode, stepIndex); err != nil {
			stepNode.SetStatus(treeview.StatusFailed)
			return err
		}
		return nil
	}

	// Execute all commands
	return e.executeCommands(ctx, stepCtx, step, stepNode, step.Commands(), stepIndex)
}

// recordStepCompletion updates execution counters and status for a completed step
func (e *Executor) recordStepCompletion(execCtx *ExecutionContext, passed bool) {
	execCtx.StepsCount++
	if passed {
		execCtx.StepsPassed++
	}
}

// evaluateStepDir evaluates and validates the step's working directory
func evaluateStepDir(execCtx *ExecutionContext) error {
	if execCtx.Step == nil || execCtx.Step.Dir == "" {
		return nil
	}
	dir, err := InterpolateString(execCtx.Step.Dir, execCtx)
	if err != nil {
		return fmt.Errorf("failed to interpolate step dir %q: %w", execCtx.Step.Dir, err)
	}
	// Resolve relative paths against the current execution directory
	if !filepath.IsAbs(dir) && execCtx.Dir != "" {
		dir = filepath.Join(execCtx.Dir, dir)
	}
	if info, statErr := os.Stat(dir); statErr != nil {
		return fmt.Errorf("step dir %q: %w", dir, statErr)
	} else if !info.IsDir() {
		return fmt.Errorf("step dir %q is not a directory", dir)
	}
	execCtx.Dir = dir
	return nil
}

// prepareStepContext creates a new execution context for a step, copying parent env and context
func (e *Executor) prepareStepContext(parentCtx *ExecutionContext, ctx context.Context, step *model.Step) (*ExecutionContext, error) {
	stepCtx := parentCtx.Copy()
	stepCtx.Context = ctx
	stepCtx.Step = step

	env := make(map[string]string)
	for k, v := range parentCtx.Env {
		env[k] = v
	}
	stepCtx.Env = env

	// Evaluate step-level working directory (overrides job dir)
	// Skip for steps with for loops - dir will be evaluated per iteration
	if step.For.IsEmpty() {
		if err := evaluateStepDir(stepCtx); err != nil {
			return nil, err
		}
	}

	return stepCtx, nil
}

// prepareIterationContext creates a new execution context for a loop iteration, overlaying iteration variables
func (e *Executor) prepareIterationContext(parentCtx *ExecutionContext, iteration model.VariableStorage) *ExecutionContext {
	iterCtx := parentCtx.Copy()
	iteration.Walk(func(k string, v any) {
		iterCtx.Variables.Set(k, v)
	})
	return iterCtx
}

// prepareIterationContextWithContext creates a new execution context for a loop iteration with context replacement
func (e *Executor) prepareIterationContextWithContext(parentCtx *ExecutionContext, ctx context.Context, iteration model.VariableStorage) (*ExecutionContext, error) {
	iterCtx := parentCtx.Copy()
	iterCtx.Context = ctx
	iteration.Walk(func(k string, v any) {
		iterCtx.Variables.Set(k, v)
	})

	// Evaluate step-level dir with iteration variables
	if err := evaluateStepDir(iterCtx); err != nil {
		return nil, err
	}

	return iterCtx, nil
}

// createIterationNode creates a new tree node for an iteration
func createIterationNode(id, name string, summarize bool) *treeview.Node {
	node := treeview.NewNode(name)
	node.SetID(id)
	node.SetStatus(treeview.StatusPending)
	node.SetSummarize(summarize)
	return node
}

// logStepSkipped marks a step as skipped and logs the skip event
func (e *Executor) logStepSkipped(execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, seqIndex int) {
	// Mark step as skipped in the tree
	stepNode.SetStatus(treeview.StatusSkipped)
	if !step.If.IsEmpty() {
		stepNode.SetIf(step.If.String())
	}

	// Get step name for logging
	stepName := step.Name
	if stepName == "" && stepNode != nil {
		stepName = stepNode.GetName()
	}

	// Get job name for ID
	jobName := ""
	if execCtx.Job != nil {
		jobName = execCtx.Job.Name
	}

	// Log SKIP event
	stepID := generateStepID(jobName, seqIndex)
	if execCtx.EventLogger != nil {
		startOffset := execCtx.EventLogger.GetElapsed()
		execCtx.EventLogger.LogExec(eventlog.ResultSkipped, stepID, stepName, startOffset, 0, nil)
	}
}

// executeStepWithForLoop handles for loop expansion and execution
// Each iteration becomes a separate execution with iteration variables overlaid on context
func (e *Executor) executeStepWithForLoop(ctx context.Context, execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, stepIndex int) error {
	// Expand the for loop to get all iterations
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: execCtx.Dir,
		DefaultEnv: execCtx.Env.Environ(),
	})
	iterations, err := ExpandFor(execCtx, func(script string) (string, error) {
		result := exec.Run(ctx, exec.ShellCommand(script))
		if !result.Success() {
			return "", NewExecError(result)
		}
		return result.Output(), nil
	})
	if err != nil {
		stepNode.SetStatus(treeview.StatusFailed)
		return fmt.Errorf("failed to expand for loop for step %q: %w", step.Name, err)
	}

	if len(iterations) == 0 {
		// Empty for loop - mark as passed
		stepNode.SetStatus(treeview.StatusPassed)
		e.recordStepCompletion(execCtx, true)
		return nil
	}

	stepNode.SetSummarize(step.Summarize)

	// Build iteration nodes as children of the step node
	iterationNodes := make([]*treeview.Node, 0, len(iterations))
	if stepNode != nil {
		// Get the command template
		var cmdTemplate string
		if step.Task == "" {
			cmdTemplate = step.String()
		}

		// Create node for each iteration with interpolated command
		for idx, iteration := range iterations {
			// Interpolate command with iteration variables
			iterCtx := e.prepareIterationContext(execCtx, iteration.Variables)

			var interpolated string
			var nodeName string
			// For task invocations, use the task name; otherwise interpolate the command
			if step.Task != "" {
				interpolated = step.Task
				nodeName = interpolated
			} else {
				var err error
				interpolated, err = InterpolateCommand(cmdTemplate, iterCtx)
				if err != nil {
					stepNode.SetStatus(treeview.StatusFailed)
					return fmt.Errorf("failed to interpolate command for iteration %d: %w", idx, err)
				}

				// Use the interpolated command as the node name
				nodeName = interpolated
			}

			// If step has a description, use that as the node name (after interpolation)
			if step.Desc != "" {
				descInterpolated, err := InterpolateCommand(step.Desc, iterCtx)
				if err == nil {
					nodeName = descInterpolated
				}
			}

			// Get job name for ID generation
			jobName := ""
			if execCtx.Job != nil {
				jobName = execCtx.Job.Name
			}

			// Generate unique ID for this iteration
			iterSeqIndex := execCtx.StepSequence + idx
			iterID := fmt.Sprintf("jobs.%s.steps.%d", jobName, iterSeqIndex)

			iterNode := createIterationNode(iterID, nodeName, step.Summarize)

			// If step has multiple commands, create child nodes for each command
			if len(step.Cmds) > 0 {
				for _, cmd := range step.Cmds {
					// Interpolate each command with iteration variables
					interpolatedCmd, err := InterpolateCommand(cmd, iterCtx)
					if err != nil {
						stepNode.SetStatus(treeview.StatusFailed)
						return fmt.Errorf("failed to interpolate command for iteration %d: %w", idx, err)
					}
					iterNode.AddChild(treeview.NewCmdNode(interpolatedCmd))
				}
			}

			// Add as child of the step node
			stepNode.AddChild(iterNode)
			iterationNodes = append(iterationNodes, iterNode)
		}
	}

	// Render tree with expanded iterations
	execCtx.Render()

	// Execute each iteration - use errgroup for detached (parallel) execution
	var eg *errgroup.Group
	if step.Detach {
		eg = new(errgroup.Group)
		eg.SetLimit(runtime.NumCPU())
	}

	var lastErr error
	var errMu sync.Mutex

	for idx, iteration := range iterations {
		// Check if context was cancelled before starting next iteration
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		idx := idx
		iteration := iteration

		executeIteration := func(iterCtx context.Context) error {
			// Create iteration context by overlaying iteration variables on parent context
			stepIterCtx, err := e.prepareIterationContextWithContext(execCtx, iterCtx, iteration.Variables)
			if err != nil {
				return fmt.Errorf("failed to prepare iteration context %d: %w", idx, err)
			}

			// Merge step-level env with interpolation
			// This needs to happen before building the command so env vars can be interpolated
			if err := MergeVariables(stepIterCtx, step.Decl); err != nil {
				return fmt.Errorf("failed to process step env for iteration %d: %w", idx, err)
			}

			// Update step node label with interpolated display label for this iteration
			if label := step.DisplayLabel(); label != "" {
				if interpolated, err := InterpolateCommand(label, stepIterCtx); err == nil {
					stepNode.SetName(interpolated)
				}
			}

			// Get the iteration sub-node
			var iterNode *treeview.Node
			if len(iterationNodes) > idx {
				iterNode = iterationNodes[idx]
			}

			// Handle task invocation or command execution
			if step.Task != "" {
				// Task invocation with loop variables
				if err := e.executeTaskStep(iterCtx, stepIterCtx, step, iterNode); err != nil {
					return err
				}
			} else {
				// Execute all commands for this iteration
				if err := e.executeCommands(iterCtx, stepIterCtx, step, iterNode, step.Commands(), stepIndex); err != nil {
					return err
				}
			}
			return nil
		}

		if step.Detach {
			// Run iterations in parallel - each gets its own cancellable context
			eg.Go(func() error {
				iterCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				if err := executeIteration(iterCtx); err != nil {
					errMu.Lock()
					if lastErr == nil {
						lastErr = err
					}
					errMu.Unlock()
				}
				return nil
			})
		} else {
			// Run iterations sequentially - break on error
			if err := executeIteration(ctx); err != nil {
				lastErr = err
				break
			}
		}
	}

	// Wait for all parallel iterations to complete
	if eg != nil {
		_ = eg.Wait()
	}

	if lastErr != nil {
		stepNode.SetStatus(treeview.StatusFailed)
		return lastErr
	}
	stepNode.SetStatus(treeview.StatusPassed)

	e.recordStepCompletion(execCtx, true)
	return nil
}

// executeStepIteration executes a single step (or iteration of a step) with the given context
func (e *Executor) executeStepIteration(ctx context.Context, stepCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, cmd string, stepIndex int) error {
	// Get step name for logging
	stepName := step.Name
	if stepName == "" && stepNode != nil {
		stepName = stepNode.GetName()
	}

	// Use the pre-assigned step sequence from the context
	seqIndex := stepCtx.StepSequence

	// Build step ID for logging
	jobName := ""
	if stepCtx.Job != nil {
		jobName = stepCtx.Job.Name
	}
	stepID := generateStepID(jobName, seqIndex)

	// Capture start offset for event log
	var startOffset float64
	if stepCtx.EventLogger != nil {
		startOffset = stepCtx.EventLogger.GetElapsed()
	}

	// Track start time for duration
	startTime := time.Now()

	// Mark step as running and render immediately to show state transition
	stepNode.SetID(stepID)
	stepNode.SetStartOffset(startOffset)
	stepNode.SetStatus(treeview.StatusRunning)
	if stepNode != nil {
		stepCtx.Render()
	}

	// Ensure output + echo label attach to the correct node (fixes output overwriting in for loops)
	originalStep := stepCtx.CurrentStep
	if stepNode != nil {
		stepCtx.CurrentStep = stepNode
	}
	defer func() { stepCtx.CurrentStep = originalStep }()

	// Handle cmds: if step has multiple commands and child nodes exist, execute each command individually
	err := e.executeCommand(ctx, stepCtx, step, cmd)

	// Calculate duration
	duration := time.Since(startTime)
	durationMs := duration.Milliseconds()

	// Update tree node status and log result
	stepNode.SetDuration(duration.Seconds())
	if err != nil {
		stepNode.SetStatus(treeview.StatusFailed)
	} else {
		stepNode.SetStatus(treeview.StatusPassed)
	}

	// Log single execution event
	if stepCtx.EventLogger != nil {
		result := eventlog.ResultPass
		if err != nil {
			result = eventlog.ResultFail
		}
		stepCtx.EventLogger.LogExec(result, stepID, stepName, startOffset, durationMs, err)
	}

	stepCtx.Render()
	return err
}

// executeTaskStep executes a task/job from within a step
// Supports both simple task invocation and for loop task invocation with loop variables
func (e *Executor) executeTaskStep(ctx context.Context, execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node) error {
	defer execCtx.Render()

	resolved, err := execCtx.Resolve(step.Task)
	if err != nil {
		stepNode.SetStatus(treeview.StatusFailed)
		return err
	}

	var (
		taskName       = resolved.Name
		targetPipeline = resolved.Pipeline
		taskJob        = resolved.Job
	)

	// Get jobs from the target pipeline for dependency resolution
	allJobs := targetPipeline.Jobs
	if len(allJobs) == 0 {
		allJobs = targetPipeline.Tasks
	}

	// Execute dependencies first (if not already completed)
	deps := GetDependencies(taskJob.DependsOn)
	for _, depName := range deps {
		if execCtx.IsJobCompleted(depName) {
			continue
		}

		if _, depExists := allJobs[depName]; !depExists {
			stepNode.SetStatus(treeview.StatusFailed)
			return fmt.Errorf("dependency %q not found for task %q", depName, taskName)
		}

		// Create a synthetic step to execute the dependency as a task
		depStep := &model.Step{Task: depName}
		if err := e.executeTaskStep(ctx, execCtx, depStep, stepNode); err != nil {
			return err
		}
	}

	// Get the existing tree node for this task
	taskJobNode := execCtx.JobNodes[taskName]
	if taskJobNode == nil {
		stepNode.SetStatus(treeview.StatusFailed)
		return fmt.Errorf("task %q node not found in tree", taskName)
	}

	taskJobNode.SetSummarize(taskJob.Summarize)
	stepNode.SetSummarize(step.Summarize)

	// Check if this step has a for loop
	if !step.For.IsEmpty() {
		// Handle task invocation with for loop
		// Don't add task node as child here - iteration nodes will be added instead
		return e.executeTaskStepWithLoop(ctx, execCtx, step, stepNode, taskJob, taskJobNode, targetPipeline)
	}

	// Add task node as child of step node so it appears expanded in the tree
	// Only for non-loop task invocations; skip if already pre-attached during tree building
	if stepNode != nil && taskJobNode != nil {
		alreadyChild := false
		for _, child := range stepNode.GetChildren() {
			if child == taskJobNode.Node {
				alreadyChild = true
				break
			}
		}
		if !alreadyChild {
			stepNode.AddChild(taskJobNode.Node)
		}
	}

	// Mark the task as running
	stepNode.SetStatus(treeview.StatusRunning)

	// Mark the task node itself as running
	taskJobNode.SetStatus(treeview.StatusRunning)
	execCtx.Render()

	// Capture task start time for logging
	var taskStartOffset float64
	if execCtx.EventLogger != nil {
		taskStartOffset = execCtx.EventLogger.GetElapsed()
	}
	taskJobNode.SetStartOffset(taskStartOffset)
	taskStartTime := time.Now()

	// Create a new execution context for the task using the task's existing tree node
	taskCtx := execCtx.Copy()
	taskCtx.Depth++
	taskCtx.Job = taskJob
	taskCtx.CurrentJob = taskJobNode
	taskCtx.Context = ctx
	taskCtx.StepSequence = 0 // Reset step counter for new job

	err = func() error {
		if err := MergeSkillVariables(taskCtx, targetPipeline.Decl); err != nil {
			return err
		}
		// Evaluate task job dir and vars with proper ordering
		if err := evaluateDirAndVars(taskCtx, taskJob, false, "task"); err != nil {
			return err
		}
		// Merge step-level vars (call-site overrides)
		// This allows step vars to be interpolated and override task defaults
		if err := MergeVariables(taskCtx, step.Decl); err != nil {
			return err
		}
		if err := ValidateJobRequirements(taskCtx, taskJob); err != nil {
			return err
		}
		if err := e.executeSteps(ctx, taskCtx, taskJob.Children()); err != nil {
			return err
		}
		return nil
	}()

	// Calculate task duration and log
	taskDuration := time.Since(taskStartTime)
	taskJobNode.SetDuration(taskDuration.Seconds())

	taskID := "jobs." + taskName
	if execCtx.EventLogger != nil {
		result := eventlog.ResultPass
		if err != nil {
			result = eventlog.ResultFail
		}
		execCtx.EventLogger.LogExec(result, taskID, taskName, taskStartOffset, taskDuration.Milliseconds(), err)
	}

	if err != nil {
		taskJobNode.SetStatus(treeview.StatusFailed)
		stepNode.SetStatus(treeview.StatusFailed)
	} else {
		taskJobNode.SetStatus(treeview.StatusPassed)
		stepNode.SetStatus(treeview.StatusPassed)
	}

	execCtx.MarkJobCompleted(taskName)
	return err
}

// executeTaskStepWithLoop executes a task multiple times via a for loop with loop variables
func (e *Executor) executeTaskStepWithLoop(ctx context.Context, execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, taskJob *model.Job, taskJobNode *treeview.TreeNode, targetPipeline *model.Pipeline) error {
	defer execCtx.Render()

	// Expand the for loop to get iteration contexts
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: execCtx.Dir,
		DefaultEnv: execCtx.Env.Environ(),
	})
	iterations, err := ExpandFor(execCtx, func(script string) (string, error) {
		result := exec.Run(ctx, exec.ShellCommand(script))
		if !result.Success() {
			return "", NewExecError(result)
		}
		return result.Output(), nil
	})
	if err != nil {
		stepNode.SetStatus(treeview.StatusFailed)
		return fmt.Errorf("failed to expand for loop: %w", err)
	}

	if len(iterations) == 0 {
		stepNode.SetStatus(treeview.StatusPassed)
		return nil
	}

	// Build iteration nodes as children of the step node (similar to executeStepWithForLoop)
	iterationNodes := make([]*treeview.TreeNode, 0, len(iterations))
	for idx, iteration := range iterations {
		// Create iteration context for node name interpolation
		iterCtx := e.prepareIterationContext(execCtx, iteration.Variables)

		// Merge step vars to get interpolated values for display
		if step.Decl != nil {
			_ = MergeVariables(iterCtx, step.Decl)
		}

		// Get job name for ID generation
		jobName := ""
		if execCtx.Job != nil {
			jobName = execCtx.Job.Name
		}

		// Generate unique ID for this iteration
		iterSeqIndex := execCtx.StepSequence + idx
		iterID := fmt.Sprintf("jobs.%s.steps.%d", jobName, iterSeqIndex)

		// Create a descriptive name showing the task and key variable values
		iterName := step.Task
		if item := iterCtx.Variables.Get("item"); item != nil {
			iterName = fmt.Sprintf("%s (item: %v)", step.Task, item)
		} else if path := iterCtx.Variables.Get("path"); path != nil {
			iterName = fmt.Sprintf("%s (path: %v)", step.Task, path)
		}

		// If step has a description, use that as the node name (after interpolation)
		if step.Desc != "" {
			descInterpolated, err := InterpolateCommand(step.Desc, iterCtx)
			if err == nil {
				iterName = descInterpolated
			}
		}

		iterNode := createIterationNode(iterID, iterName, step.Summarize)

		// Add as child of the step node
		stepNode.AddChild(iterNode)

		// Create a TreeNode wrapper for the iteration
		iterTreeNode := &treeview.TreeNode{Node: iterNode}
		iterationNodes = append(iterationNodes, iterTreeNode)
	}

	// Render tree with expanded iterations
	execCtx.Render()

	// Execute task for each iteration - use errgroup for detached (parallel) execution
	var eg *errgroup.Group
	if step.Detach {
		eg = new(errgroup.Group)
		eg.SetLimit(runtime.NumCPU())
	}

	var lastErr error
	var errMu sync.Mutex

	for idx, iter := range iterations {
		// Check if context was cancelled before starting next iteration
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		idx := idx
		iter := iter

		executeIteration := func(iterRunCtx context.Context) error {
			iterTreeNode := iterationNodes[idx]

			// Create execution context for this iteration with loop variables
			iterCtx := execCtx.Copy()
			iter.Variables.Walk(func(k string, v any) {
				iterCtx.Variables.Set(k, v)
			})
			iterCtx.Job = taskJob
			iterCtx.CurrentJob = iterTreeNode // Use iteration-specific node
			iterCtx.Context = iterRunCtx
			iterCtx.StepSequence = 0 // Reset step counter for each iteration

			// Mark iteration as running
			iterTreeNode.SetStatus(treeview.StatusRunning)
			execCtx.Render()

			if err := MergeSkillVariables(iterCtx, targetPipeline.Decl); err != nil {
				iterTreeNode.SetStatus(treeview.StatusFailed)
				return err
			}

			// Evaluate task job dir and vars with proper ordering
			if err := evaluateDirAndVars(iterCtx, taskJob, false, "task"); err != nil {
				iterTreeNode.SetStatus(treeview.StatusFailed)
				return err
			}

			// Merge step-level vars (call-site overrides) with iteration context
			// This allows step vars like `path: $(dirname "${{item}}")` to be interpolated
			if err := MergeVariables(iterCtx, step.Decl); err != nil {
				iterTreeNode.SetStatus(treeview.StatusFailed)
				return err
			}

			// Validate job requirements (loop variables should satisfy requires)
			if err := ValidateJobRequirements(iterCtx, taskJob); err != nil {
				iterTreeNode.SetStatus(treeview.StatusFailed)
				return err
			}

			// Execute the task job steps with the iteration's own context
			if err := e.executeSteps(iterRunCtx, iterCtx, taskJob.Children()); err != nil {
				iterTreeNode.SetStatus(treeview.StatusFailed)
				return err
			}

			// Mark iteration as passed
			iterTreeNode.SetStatus(treeview.StatusPassed)
			return nil
		}

		if step.Detach {
			// Run iterations in parallel - each gets its own cancellable context
			eg.Go(func() error {
				iterCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				if err := executeIteration(iterCtx); err != nil {
					errMu.Lock()
					if lastErr == nil {
						lastErr = err
					}
					errMu.Unlock()
				}
				return nil
			})
		} else {
			// Run iterations sequentially - break on error
			if err := executeIteration(ctx); err != nil {
				lastErr = err
				break
			}
		}
	}

	// Wait for all parallel iterations to complete
	if eg != nil {
		_ = eg.Wait()
	}

	// Update parent node statuses based on results
	if lastErr != nil {
		taskJobNode.SetStatus(treeview.StatusFailed)
		stepNode.SetStatus(treeview.StatusFailed)
		return lastErr
	}

	// Mark task and step as passed
	taskJobNode.SetStatus(treeview.StatusPassed)
	stepNode.SetStatus(treeview.StatusPassed)

	return nil
}

// interpolateVariables interpolates all string variables in a map using $(exec) and ${{ var }} syntax.
// Non-string values are passed through unchanged.
// Variables are evaluated in dependency order using topological sort.
// Returns the interpolated map or an error if interpolation fails.
func interpolateVariables(ctx *ExecutionContext, vars map[string]any) (map[string]any, error) {
	if vars == nil {
		return nil, nil
	}

	if ctx == nil {
		return vars, nil
	}

	// Build dependency graph
	deps := make(map[string][]string)
	for k, v := range vars {
		if strVal, ok := v.(string); ok {
			deps[k] = extractVariableDependencies(strVal, vars)
		} else {
			deps[k] = nil
		}
	}

	// Topological sort
	order, err := topologicalSort(deps)
	if err != nil {
		return nil, err
	}

	// Create a working context that accumulates resolved variables
	workCtx := &ExecutionContext{
		Variables: ctx.Variables.Clone(),
		Env:       ctx.Env,
		Dir:       ctx.Dir,
	}

	result := make(map[string]any)
	for _, k := range order {
		v := vars[k]
		if strVal, ok := v.(string); ok {
			interpolated, err := InterpolateString(strVal, workCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate variable %q: %w", k, err)
			}
			result[k] = interpolated
			workCtx.Variables.Set(k, interpolated)
		} else {
			result[k] = v
			workCtx.Variables.Set(k, v)
		}
	}
	return result, nil
}

// IsEchoCommand checks if a command is a bare echo command.
func IsEchoCommand(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	return strings.HasPrefix(trimmed, "echo ") && !strings.Contains(trimmed, "\n")
}

// evaluateEchoCommand executes an echo command and returns its output for use as a label
func evaluateEchoCommand(ctx context.Context, env Env, dir string, cmd string) (string, error) {
	exec := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: dir,
		DefaultEnv: env.Environ(),
	})
	result := exec.Run(ctx, exec.ShellCommand(cmd))
	if !result.Success() {
		return "", NewExecError(result)
	}
	return strings.TrimSpace(result.Output()), nil
}

// executeCommand runs a single command with interpolation and respects context timeout
func (e *Executor) executeCommand(ctx context.Context, execCtx *ExecutionContext, step *model.Step, cmd string) error {
	// Interpolate the command
	interpolated, err := InterpolateCommand(cmd, execCtx)
	if err != nil {
		return fmt.Errorf("interpolation failed: %w", err)
	}

	// Check if context is already cancelled
	if ctx != nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("command execution cancelled or timed out: %w", ctx.Err())
		default:
		}
	}

	// Determine if interactive mode should be used (live streaming with stdin)
	// Check step interactive flag first, then job interactive flag
	isInteractive := step.Interactive || (execCtx.Job != nil && execCtx.Job.Interactive)

	// Determine if output should be captured for display with tree indentation
	// Check step passthru flag first, then job passthru flag
	shouldPassthru := step.Passthru || (execCtx.Job != nil && execCtx.Job.Passthru)

	// Determine TTY allocation: Job.TTY is authoritative, otherwise use Step.TTY
	useTTY := step.TTY || (execCtx.Job != nil && execCtx.Job.TTY)

	// Track execution for logging
	startTime := time.Now()
	var startOffset float64
	if execCtx.EventLogger != nil {
		startOffset = execCtx.EventLogger.GetElapsed()
	}

	// Execute the command
	executor := psexec.NewWithOptions(&psexec.Options{
		DefaultDir: execCtx.Dir,
		DefaultEnv: execCtx.Env.Environ(),
	})
	shellCmd := executor.ShellCommand(interpolated)

	var writer *LineCapturingWriter
	var result psexec.Result
	if isInteractive {
		shellCmd.Interactive = true
		result = executor.Run(ctx, shellCmd)
		execCtx.Display.Invalidate()
	} else if shouldPassthru && execCtx.CurrentStep != nil {
		// If passthru is enabled, capture output to the node for display with tree indentation
		writer = NewLineCapturingWriter()
		shellCmd.Stdout = writer
		shellCmd.Stderr = writer
		shellCmd.UsePTY = useTTY
		result = executor.Run(ctx, shellCmd)
	} else {
		result = executor.Run(ctx, shellCmd)
	}

	// Log command execution
	durationMs := time.Since(startTime).Milliseconds()
	if execCtx.EventLogger != nil {
		exitCode := result.ExitCode()
		errMsg := ""
		if !result.Success() {
			errMsg = result.ErrorOutput()
			if errMsg == "" && result.Err() != nil {
				errMsg = result.Err().Error()
			}
		}
		stepID := ""
		if execCtx.CurrentStep != nil {
			stepID = execCtx.CurrentStep.ID
		}
		output := result.Output()
		if writer != nil {
			output = writer.String()
		}
		execCtx.EventLogger.LogCommand(eventlog.LogEntry{
			Type:       eventlog.EventTypeStep,
			ID:         stepID,
			Command:    interpolated,
			Dir:        execCtx.Dir,
			Output:     output,
			Error:      errMsg,
			ExitCode:   exitCode,
			Start:      startOffset,
			DurationMs: durationMs,
		})
	}

	if !result.Success() {
		return NewExecError(result)
	}

	// Set output on node only after command completes successfully
	if execCtx.CurrentStep != nil {
		// For echo commands, update the step node label with the output
		if IsEchoCommand(interpolated) {
			echoOutput, echoErr := evaluateEchoCommand(ctx, execCtx.Env, execCtx.Dir, interpolated)
			if echoErr == nil && echoOutput != "" {
				execCtx.CurrentStep.Name = echoOutput
			}
		} else if writer != nil {
			rawOutput := writer.String()
			lines, sanitizeErr := Sanitize(rawOutput)
			if sanitizeErr != nil {
				return fmt.Errorf("failed to sanitize output: %w", sanitizeErr)
			}
			if len(lines) > 0 {
				execCtx.CurrentStep.SetOutput(lines)
			}
		}
	}

	return nil
}
