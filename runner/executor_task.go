package runner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/treeview"
)

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

	execCtx.EmitProgress(JobProgressEvent{
		JobName:   taskName,
		Parents:   execCtx.Parents,
		Status:    JobProgressRunning,
		StartedAt: taskStartTime,
	})

	// Create a new execution context for the task using the task's existing tree node
	taskCtx := execCtx.Copy()
	taskCtx.Depth++
	taskCtx.Job = taskJob
	taskCtx.CurrentJob = taskJobNode
	taskCtx.Context = ctx
	taskCtx.StepSequence = 0 // Reset step counter for new job
	taskCtx.Parents = append(append([]string(nil), execCtx.Parents...), taskName)

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

		execCtx.EmitProgress(JobProgressEvent{
			JobName:   taskName,
			Parents:   execCtx.Parents,
			Status:    JobProgressFailed,
			StartedAt: taskStartTime,
			Duration:  taskDuration,
			Err:       err,
		})
	} else {
		taskJobNode.SetStatus(treeview.StatusPassed)
		stepNode.SetStatus(treeview.StatusPassed)

		execCtx.EmitProgress(JobProgressEvent{
			JobName:   taskName,
			Parents:   execCtx.Parents,
			Status:    JobProgressPassed,
			StartedAt: taskStartTime,
			Duration:  taskDuration,
		})
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
