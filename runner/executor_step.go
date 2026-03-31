package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/treeview"
)

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

		if stepNode != nil {
			// Update status to running and re-render to show the transition
			stepNode.SetStatus(treeview.StatusRunning)

			if err := e.executeStepWithNode(ctx, execCtx, step, stepNode); err != nil {
				return err
			}
		} else {
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

// executeStepWithForLoop handles for loop expansion and execution.
// Each iteration becomes a separate execution with iteration variables overlaid on context.
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
