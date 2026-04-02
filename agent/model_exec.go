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
func (m Model) runPipeline(task *model.ResolvedTask) tea.Cmd {
	return func() tea.Msg {
		jobName := task.Job.Name
		start := time.Now()

		ctx := context.Background()
		err := runner.RunPipeline(ctx, task.Pipeline, runner.PipelineOptions{
			Jobs:         []string{jobName},
			Silent:       true,
			Debug:        m.agent.Options().Debug,
			AllPipelines: m.agent.Pipelines(),
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
func (m Model) runAutofixPipeline(originalTask, fixTask *model.ResolvedTask) tea.Cmd {
	return func() tea.Msg {
		jobName := fixTask.Job.Name
		start := time.Now()

		ctx := context.Background()
		err := runner.RunPipeline(ctx, fixTask.Pipeline, runner.PipelineOptions{
			Jobs:         []string{jobName},
			Silent:       true,
			Debug:        m.agent.Options().Debug,
			AllPipelines: m.agent.Pipelines(),
		})

		return AutofixDoneMsg{
			OriginalTask: originalTask,
			Err:          err,
			Duration:     time.Since(start),
		}
	}
}

// runShellCommand runs a shell command and captures output.
func (m Model) runShellCommand(command string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		cmd := exec.Command("sh", "-c", command)
		cmd.Dir = m.cwd
		out, err := cmd.CombinedOutput()

		exitCode := 0
		if err != nil {
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
