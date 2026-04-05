package agent

import (
	"context"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// getFixTask returns the fix task for a given task, or nil if none exists.
func (m Model) getFixTask(task *model.ResolvedTask) *model.ResolvedTask {
	if task == nil || task.Pipeline == nil || task.Pipeline.ID == "" {
		return nil
	}
	fixName := task.Pipeline.ID + ":fix"
	fixTask, err := m.agent.Resolver().Resolve(fixName)
	if err != nil {
		return nil
	}
	return fixTask
}

// runPipeline executes the task silently and returns the result with duration.
// Silent mode suppresses tree display; only error messages are captured.
// The progress channel receives job lifecycle events during execution.
// The context allows cancellation via Escape or Ctrl+C.
func (m Model) runPipeline(ctx context.Context, task *model.ResolvedTask, progressCh chan runner.JobProgressEvent) tea.Cmd {
	return func() tea.Msg {
		defer close(progressCh)

		jobName := task.Job.Name
		start := time.Now()

		err := runner.RunPipeline(ctx, task.Pipeline, runner.PipelineOptions{
			Jobs:         []string{jobName},
			Silent:       true,
			Debug:        m.agent.Options().Debug,
			AllPipelines: m.agent.Pipelines(),
			Progress: runner.ProgressObserverFunc(func(ev runner.JobProgressEvent) {
				progressCh <- ev
			}),
		})

		return ExecutionDoneMsg{
			Task:     task,
			Err:      err,
			Duration: time.Since(start),
		}
	}
}

// runAutofixPipeline runs the fix task and then signals completion.
// Silent mode suppresses tree display; only error messages are captured.
// The context allows cancellation via Escape or Ctrl+C.
func (m Model) runAutofixPipeline(ctx context.Context, originalTask, fixTask *model.ResolvedTask, progressCh chan runner.JobProgressEvent) tea.Cmd {
	return func() tea.Msg {
		defer close(progressCh)

		jobName := fixTask.Job.Name
		start := time.Now()

		err := runner.RunPipeline(ctx, fixTask.Pipeline, runner.PipelineOptions{
			Jobs:         []string{jobName},
			Silent:       true,
			Debug:        m.agent.Options().Debug,
			AllPipelines: m.agent.Pipelines(),
			Progress: runner.ProgressObserverFunc(func(ev runner.JobProgressEvent) {
				progressCh <- ev
			}),
		})

		return AutofixDoneMsg{
			OriginalTask: originalTask,
			Err:          err,
			Duration:     time.Since(start),
		}
	}
}

// runShellCommand runs a shell command and captures output.
// The context allows cancellation via Escape or Ctrl+C.
func (m Model) runShellCommand(ctx context.Context, command string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = m.cwd
		out, err := cmd.CombinedOutput()

		exitCode := 0
		if err != nil {
			// Check if it was a context cancellation
			if ctx.Err() == context.Canceled {
				return ShellDoneMsg{
					Command:  command,
					Output:   string(out),
					Err:      context.Canceled,
					ExitCode: -1,
					Duration: time.Since(start),
				}
			}
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		return ShellDoneMsg{
			Command:  command,
			Output:   string(out),
			Err:      err,
			ExitCode: exitCode,
			Duration: time.Since(start),
		}
	}
}
