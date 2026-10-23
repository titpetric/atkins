package runner

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/treeview"
)

// prepareIterationContext creates a new execution context for a loop iteration, overlaying iteration variables
func (e *Executor) prepareIterationContext(parentCtx *ExecutionContext, iteration model.VariableStorage) *ExecutionContext {
	iterCtx := parentCtx.Copy()
	// An iteration is a step execution of its own, with a memo of its own
	// substitutions: the previous iteration's results don't carry over.
	iterCtx.substitutions = newSubstitutionCache()
	iteration.Walk(func(k string, v any) {
		iterCtx.Variables.Set(k, v)
	})
	return iterCtx
}

// prepareIterationContextWithContext creates a new execution context for a loop iteration with context replacement
func (e *Executor) prepareIterationContextWithContext(parentCtx *ExecutionContext, ctx context.Context, iteration model.VariableStorage) (*ExecutionContext, error) {
	iterCtx := parentCtx.Copy()
	iterCtx.Context = ctx
	// An iteration is a step execution of its own; see
	// prepareIterationContext.
	iterCtx.substitutions = newSubstitutionCache()
	iteration.Walk(func(k string, v any) {
		iterCtx.Variables.Set(k, v)
	})

	// Resolve step-level dir with iteration variables
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

// executeStepWithForLoop handles for loop expansion and execution.
// Each iteration becomes a separate execution with iteration variables overlaid on context.
func (e *Executor) executeStepWithForLoop(ctx context.Context, execCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, stepIndex int) error {
	// Step vars may hold the collection the loop iterates. Keep them lazy so
	// only vars the loop references resolve before iteration variables
	// exist, and cache those values for all iterations.
	if err := prepareStepLoopVariables(execCtx, step); err != nil {
		return fmt.Errorf("failed to prepare for loop: %w", err)
	}

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

	// The runner interpolates an iteration twice: once below to name its node
	// before anything runs, and once again when it runs. One memo per
	// iteration, held here, is what keeps a `$(...)` in the command to
	// one execution across both.
	memos := make([]*substitutionCache, len(iterations))
	for idx := range memos {
		memos[idx] = newSubstitutionCache()
	}

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
			iterCtx.substitutions = memos[idx]

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

			// If step has more than one command, create child nodes for each command
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

	stepStartTime := time.Now()
	defer func() {
		stepNode.SetDuration(time.Since(stepStartTime).Seconds())
	}()

	for idx, iteration := range iterations {
		// Check if the context ended before starting next iteration
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
			stepIterCtx.substitutions = memos[idx]

			// Merge step-level env with interpolation
			// This happens before building the command so interpolation can use env vars
			iterationDecl := stepDeclWithoutResolvedVars(step.Decl, iteration.Variables)
			if err := MergeVariables(stepIterCtx, iterationDecl); err != nil {
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

// prepareStepLoopVariables overlays step vars as lazy values. Existing parent
// vars remain visible, while step vars keep the usual precedence.
func prepareStepLoopVariables(ctx *ExecutionContext, step *model.Step) error {
	if step.Decl == nil {
		return nil
	}

	// Resolve the first source's unified declaration closure before installing
	// the remaining lazy vars. Later iterator sources may depend on iterator
	// variables, so they remain pending until expansion reaches them.
	resolvedVars := map[string]bool{}
	resolvedEnv := map[string]bool{}
	if len(step.For) > 0 {
		items, _, _, _, err := parseForPattern(string(step.For[0]))
		if err != nil {
			return err
		}
		var envVars map[string]any
		if step.Decl.Env != nil {
			envVars = step.Decl.Env.Vars
		}
		roots := extractUnifiedDependencies(items, step.Decl.Vars, envVars)
		if name, direct := directForVariable(items); direct {
			if _, ok := step.Decl.Vars[name]; ok {
				roots = append(roots, nodePrefixVar+name)
			} else if _, ok := envVars[name]; ok {
				roots = append(roots, nodePrefixEnv+name)
			}
		}
		if len(roots) > 0 {
			r, err := newResolver(ctx, step.Decl)
			if err != nil {
				return err
			}
			resolvedVars, resolvedEnv, err = r.resolveRequiredInto(ctx, roots)
			if err != nil {
				return err
			}
		}
	}

	pending := make(map[string]any)
	for k, v := range step.Decl.Vars {
		if !resolvedVars[k] {
			pending[k] = v
		}
	}
	lazyVars := NewContextVariablesWithResolver(pending, func(s string) (string, error) {
		return InterpolateString(s, ctx)
	})
	ctx.Variables.Walk(func(k string, v any) {
		if _, overridden := pending[k]; !overridden {
			lazyVars.Set(k, v)
		}
	})
	ctx.Variables = lazyVars
	lazyVars.loopResolvedEnv = resolvedEnv
	return nil
}

// stepDeclWithoutResolvedVars leaves vars resolved during loop expansion out
// of per-iteration merging. Their cached values are already in each iteration;
// unresolved vars still resolve once iteration variables exist.
func stepDeclWithoutResolvedVars(decl *model.Decl, variables model.VariableStorage) *model.Decl {
	lazyVars, ok := variables.(*ContextVariables)
	if decl == nil || !ok {
		return decl
	}

	copyDecl := *decl
	copyDecl.Vars = make(map[string]any, len(decl.Vars))
	for k, v := range decl.Vars {
		if !lazyVars.IsResolved(k) {
			copyDecl.Vars[k] = v
		}
	}
	if decl.Env != nil {
		envCopy := *decl.Env
		envCopy.Vars = make(map[string]any, len(decl.Env.Vars))
		for k, v := range decl.Env.Vars {
			// Values already present in the expansion environment hold cached source
			// dependencies; other env declarations remain per-iteration.
			if !lazyVars.loopResolvedEnv[k] {
				envCopy.Vars[k] = v
			}
		}
		copyDecl.Env = &envCopy
	}
	return &copyDecl
}
