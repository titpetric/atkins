package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/psexec"
	"github.com/titpetric/atkins/treeview"
)

// executeCommands executes a list of commands, updating child nodes if available.
// Returns the last error encountered (continues on errors to collect all failures).
func (e *Executor) executeCommands(ctx context.Context, stepCtx *ExecutionContext, step *model.Step, stepNode *treeview.Node, commands []string, stepIndex int) error {
	if len(commands) == 0 {
		return nil
	}

	// If step has more than one command, update child nodes individually
	var cmdNodes []*treeview.Node
	if stepNode != nil {
		cmdNodes = stepNode.GetChildren()
	}

	var lastErr error
	startTime := time.Now()
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
		stepNode.SetDuration(time.Since(startTime).Seconds())
		if lastErr != nil {
			stepNode.SetStatus(treeview.StatusFailed)
		} else {
			stepNode.SetStatus(treeview.StatusPassed)
		}
	}

	return lastErr
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

	// cmds: if step has more than one command and child nodes exist, execute each command individually
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

// executeCommand runs a single command with interpolation and respects context timeout
func (e *Executor) executeCommand(ctx context.Context, execCtx *ExecutionContext, step *model.Step, cmd string) error {
	// An exec: step runs its argv without a shell; the display string
	// in cmd is only for the tree and the log.
	var argv []string
	if len(step.Exec) > 0 {
		var err error
		argv, err = execArgv(execCtx, step)
		if err != nil {
			return err
		}
		cmd = (&model.Step{Exec: argv}).ExecString()
	}

	// Interpolate the command
	interpolated := cmd
	if argv == nil {
		var err error
		interpolated, err = InterpolateCommand(cmd, execCtx)
		if err != nil {
			return fmt.Errorf("interpolation failed: %w", err)
		}
	}

	// Check if context is already cancelled
	if ctx != nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("command execution cancelled or timed out: %w", ctx.Err())
		default:
		}
	}

	// Determine if interactive mode applies (live streaming with stdin)
	// Check step interactive flag first, then job interactive flag
	isInteractive := step.Interactive || (execCtx.Job != nil && execCtx.Job.Interactive)

	// Determine whether to capture output for display with tree indentation
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
	if argv != nil {
		shellCmd = psexec.NewCommand(argv[0], argv[1:]...)
	}

	var writer *LineCapturingWriter
	var result psexec.Result
	if isInteractive {
		shellCmd.Interactive = true
		result = executor.Run(ctx, shellCmd)
		execCtx.Display.Invalidate()
	} else if shouldPassthru && execCtx.CurrentStep != nil {
		// With passthru on, capture output to the node for display with tree indentation
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

	// Set output on node only after command completes successfully.
	// Both branches reuse the output captured by the run above; running the
	// command a second time here would repeat its side effects.
	if execCtx.CurrentStep != nil {
		captured := result.Output()
		if writer != nil {
			captured = writer.String()
		}

		// For echo commands, update the step node label with the output
		if argv == nil && IsEchoCommand(interpolated) {
			if echoOutput := strings.TrimSpace(captured); echoOutput != "" {
				execCtx.CurrentStep.Name = echoOutput
			}
		} else if writer != nil {
			lines, sanitizeErr := Sanitize(captured)
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

// execArgv builds the argv of an exec: step. It interpolates each
// exec: item on its own, and each item stays one argument. A job with args: true
// then gets the command line arguments appended as they are, with no
// interpolation, globbing or word splitting.
func execArgv(execCtx *ExecutionContext, step *model.Step) ([]string, error) {
	argv := make([]string, 0, len(step.Exec)+len(execCtx.Args))
	for _, item := range step.Exec {
		interpolated, err := InterpolateString(item, execCtx)
		if err != nil {
			return nil, fmt.Errorf("interpolation failed: %w", err)
		}
		argv = append(argv, interpolated)
	}
	if argv[0] == "" {
		return nil, fmt.Errorf("exec: names no program")
	}
	if execCtx.Job != nil && execCtx.Job.Args {
		argv = append(argv, execCtx.Args...)
	}
	return argv, nil
}

// IsEchoCommand checks if a command is a bare echo command.
func IsEchoCommand(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	return strings.HasPrefix(trimmed, "echo ") && !strings.Contains(trimmed, "\n")
}

// interpolateVariables interpolates all string variables in a map using $(exec) and ${{ var }} syntax.
// Non-string values pass through unchanged.
// It resolves variables in dependency order using topological sort.
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
