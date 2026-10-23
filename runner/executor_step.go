package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
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
				// so that an error cancels only this tree
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

	// Second pass: execute deferred steps after all detached steps finish
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
	// and merge inside the iteration context
	if step.For.IsEmpty() {
		if err := MergeVariables(stepCtx, step.Decl); err != nil {
			stepNode.SetStatus(treeview.StatusFailed)
			return fmt.Errorf("failed to process step env: %w", err)
		}
	}

	stepCtx.CurrentStep = stepNode

	// Check if condition
	shouldRun, err := EvaluateIf(stepCtx)
	if err != nil {
		// If condition evaluation fails, skip the step
		stepNode.SetStatus(treeview.StatusSkipped)
		return fmt.Errorf("failed to evaluate if condition for step %q: %w", step.Name, err)
	}

	if !shouldRun {
		seqIndex := execCtx.NextStepIndex()
		e.logStepSkipped(stepCtx, step, stepNode, seqIndex)
		return nil
	}

	// Task invocations own their loop expansion.
	if step.Task != "" {
		stepNode.SetStatus(treeview.StatusRunning)
		return e.executeTaskStep(ctx, stepCtx, step, stepNode)
	}

	// Handle command for loop expansion.
	if !step.For.IsEmpty() {
		return e.executeStepWithForLoop(ctx, stepCtx, step, stepNode, 0)
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
	// and merge inside the iteration context
	if step.For.IsEmpty() {
		if err := MergeVariables(stepCtx, step.Decl); err != nil {
			stepNode.SetStatus(treeview.StatusFailed)
			return fmt.Errorf("failed to process step env: %w", err)
		}
	}

	// Interpolate step display label with current context.
	//
	// A step with a for loop skips it: the label still holds the loop's
	// own variables, and each iteration sets the node's name again from
	// its own context below. Rendering it here would execute the
	// substitutions in it for a name nothing keeps.
	if label := step.DisplayLabel(); label != "" && step.For.IsEmpty() {
		if interpolated, err := InterpolateCommand(label, stepCtx); err == nil {
			stepNode.SetName(interpolated)
		}
	}

	// Check if condition
	shouldRun, err := EvaluateIf(stepCtx)
	if err != nil {
		// If condition evaluation fails, skip the step
		stepNode.SetStatus(treeview.StatusSkipped)
		return fmt.Errorf("failed to evaluate if condition for step %q: %w", step.Name, err)
	}

	if !shouldRun {
		e.logStepSkipped(stepCtx, step, stepNode, seqIndex)
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

	// One step, one set of answers: the passes that interpolate this
	// step share them, and the next step asks again.
	stepCtx.substitutions = newSubstitutionCache()

	env := make(map[string]string)
	for k, v := range parentCtx.Env {
		env[k] = v
	}
	stepCtx.Env = env

	// Resolve step-level working directory (overrides job dir)
	// Skip for steps with for loops - each iteration resolves dir
	if step.For.IsEmpty() {
		if err := evaluateStepDir(stepCtx); err != nil {
			return nil, err
		}
	}

	return stepCtx, nil
}

// logStepSkipped marks a step as skipped and logs the skip event
func (e *Executor) logStepSkipped(execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, seqIndex int) {
	// Mark step as skipped in the tree
	stepNode.SetStatus(treeview.StatusSkipped)
	if !step.If.IsEmpty() {
		stepNode.SetIf(interpolateIfDisplay(step.If.String(), execCtx))
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
