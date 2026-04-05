package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/titpetric/atkins/agent/aliases"
	"github.com/titpetric/atkins/agent/router"
	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// Output abstracts output operations for both interactive and non-interactive modes.
type Output interface {
	// Info prints informational output.
	Info(text string)
	// Error prints error output.
	Error(text string)
	// Prompt echoes the user's input.
	Prompt(text string)
	// CommandOutput prints command output (stdout/stderr).
	CommandOutput(text string)
}

// StdOutput implements Output for non-interactive mode (stdout/stderr).
type StdOutput struct {
	Out io.Writer
	Err io.Writer
}

// NewStdOutput creates a new stdout-based output.
func NewStdOutput() *StdOutput {
	return &StdOutput{Out: os.Stdout, Err: os.Stderr}
}

func (o *StdOutput) Info(text string) { fmt.Fprintln(o.Out, text) }

func (o *StdOutput) Error(text string) { fmt.Fprintln(o.Err, text) }

func (o *StdOutput) Prompt(text string) { /* no-op for -x mode */ }

func (o *StdOutput) CommandOutput(text string) { fmt.Fprint(o.Out, text) }

// Executor handles route execution with consistent output.
type Executor struct {
	agent   *Agent
	router  *router.Router
	out     Output
	workDir string
	ctx     context.Context
}

// NewExecutor creates a new executor.
func NewExecutor(ctx context.Context, agent *Agent, rtr *router.Router, out Output) *Executor {
	return &Executor{
		agent:   agent,
		router:  rtr,
		out:     out,
		workDir: agent.WorkDir(),
		ctx:     ctx,
	}
}

// ExecuteRoute handles a routed command and returns an error if execution failed.
func (e *Executor) ExecuteRoute(route *router.Route) error {
	switch route.Type {
	case router.RouteTask, router.RouteAlias:
		if route.Resolved == nil {
			return fmt.Errorf("could not resolve: %s", route.Raw)
		}
		return e.ExecuteTask(route.Resolved)

	case router.RouteMultiTask:
		for _, task := range route.Tasks {
			if err := e.ExecuteTask(task); err != nil {
				return err
			}
		}
		return nil

	case router.RouteShell:
		return e.ExecuteShell(route.ShellCmd)

	case router.RouteHelp:
		e.out.Info(UsageText())
		return nil

	case router.RouteSlash:
		return e.handleSlashCommand(route)

	case router.RouteQuit:
		return nil

	case router.RouteGreeting:
		e.out.Info(route.Greeting)
		return nil

	case router.RouteFortune:
		e.out.Info(route.Fortune)
		return nil

	case router.RouteCorrection:
		e.router.Aliases().Add(route.Phrase, route.AliasTask)
		e.out.Info(fmt.Sprintf("Got it! \"%s\" will now run %s", route.Phrase, route.AliasTask))
		return nil

	case router.RouteConfirm:
		e.out.Info(fmt.Sprintf("Did you mean %s?", route.Suggestion))
		e.out.Info(fmt.Sprintf("Run: atkins -x \"%s\"", route.Suggestion))
		return fmt.Errorf("unknown command: %s", route.Original)

	default:
		// RouteUnknown
		if route.Ambiguous && (len(route.Matches) > 0 || len(route.HistMatches) > 0) {
			if len(route.Matches) > 0 {
				e.out.Info("Matching skills:")
				for _, match := range route.Matches {
					e.out.Info("  " + match)
				}
			}
			if len(route.HistMatches) > 0 {
				if len(route.Matches) > 0 {
					e.out.Info("")
				}
				e.out.Info("From shell history:")
				for _, h := range route.HistMatches {
					status := "exit 0"
					if h.ExitCode != 0 {
						status = fmt.Sprintf("exit %d", h.ExitCode)
					}
					e.out.Info(fmt.Sprintf("  $ %s (%s)", h.Command, status))
				}
			}
			e.out.Info("\nBe more specific or use $ <command>")
			return nil
		}
		return fmt.Errorf("unknown command: %s", route.Raw)
	}
}

// ExecuteTask runs a resolved task.
func (e *Executor) ExecuteTask(task *model.ResolvedTask) error {
	start := time.Now()
	err := runner.RunPipeline(e.ctx, task.Pipeline, runner.PipelineOptions{
		Jobs:         []string{task.Job.Name},
		Silent:       true,
		Debug:        e.agent.options.Debug,
		AllPipelines: e.agent.pipelines,
	})
	dur := time.Since(start)

	if err != nil {
		e.out.Error(fmt.Sprintf("%s %s %s",
			colors.BrightRed("✗"), task.Name,
			colors.BrightRed("FAIL")+" "+colors.Dim(fmt.Sprintf("%.2fs", dur.Seconds()))))
		return err
	}
	e.out.Info(fmt.Sprintf("%s %s %s",
		colors.BrightGreen("✓"), task.Name,
		colors.BrightGreen("OK")+" "+colors.Dim(fmt.Sprintf("%.2fs", dur.Seconds()))))
	return nil
}

// ExecuteShell runs a shell command.
func (e *Executor) ExecuteShell(command string) error {
	cmd := exec.CommandContext(e.ctx, "sh", "-c", command)
	cmd.Dir = e.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Record in shell history
	e.router.ShellHistory().Add(command, exitCode, dur, e.workDir)

	return err
}

// handleSlashCommand handles slash commands in non-interactive mode.
func (e *Executor) handleSlashCommand(route *router.Route) error {
	switch route.Command {
	case "list", "l", "ls", "skills":
		e.printSkillList()
		return nil
	case "help", "h", "?":
		e.out.Info(UsageText())
		return nil
	case "aliases", "alias":
		e.printAliases(e.router.Aliases())
		return nil
	default:
		return fmt.Errorf("slash command /%s is only available in interactive mode", route.Command)
	}
}

// printSkillList prints available skills in the same format as `atkins -l`.
func (e *Executor) printSkillList() {
	pipelines := e.agent.Pipelines()
	if len(pipelines) == 0 {
		e.out.Info("No skills available")
		return
	}

	output := runner.ListPipelines(pipelines)
	// Print line by line, trimming trailing newline
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		e.out.Info(line)
	}
}

// printAliases prints defined aliases.
func (e *Executor) printAliases(aliasStore *aliases.Aliases) {
	if len(aliasStore.Aliases) == 0 {
		e.out.Info("No aliases defined.")
		e.out.Info("")
		e.out.Info("Teach an alias with:")
		e.out.Info("  alias <phrase> to <command>")
		return
	}

	e.out.Info("Defined aliases:")
	for _, alias := range aliasStore.Aliases {
		e.out.Info(fmt.Sprintf("  %s → %s", alias.Phrase, alias.Prompt))
	}
}

// ShellResult contains the result of a shell command execution.
type ShellResult struct {
	Command  string
	Output   string
	ExitCode int
	Duration time.Duration
	Err      error
}

// ExecuteShellCapture runs a shell command and captures output.
func (e *Executor) ExecuteShellCapture(command string) ShellResult {
	cmd := exec.CommandContext(e.ctx, "sh", "-c", command)
	cmd.Dir = e.workDir

	start := time.Now()
	output, err := cmd.CombinedOutput()
	dur := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Record in shell history
	e.router.ShellHistory().Add(command, exitCode, dur, e.workDir)

	return ShellResult{
		Command:  command,
		Output:   strings.TrimRight(string(output), "\n"),
		ExitCode: exitCode,
		Duration: dur,
		Err:      err,
	}
}
