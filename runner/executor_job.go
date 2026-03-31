package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/treeview"
)

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
		if err := evaluateDirAndVarsSkipDir(execCtx, job); err != nil {
			return err
		}
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

// evaluateDirAndVarsSkipDir merges vars and env from a job without evaluating
// job.Dir. This is used when a job has a for loop, where dir evaluation is
// deferred to each iteration (it may reference loop variables).
func evaluateDirAndVarsSkipDir(ctx *ExecutionContext, job *model.Job) error {
	// Set up lazy evaluation for vars
	if job.Decl != nil && job.Decl.Vars != nil {
		lazyVars := NewContextVariablesWithResolver(job.Decl.Vars, func(s string) (string, error) {
			return InterpolateString(s, ctx)
		})
		ctx.Variables.Walk(func(k string, v any) {
			lazyVars.Set(k, v)
		})
		ctx.Variables = lazyVars
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
