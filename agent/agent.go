package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// Options configures agent behavior.
type Options struct {
	Debug   bool
	Verbose bool
	Jail    bool
}

// Agent manages the interactive REPL session.
type Agent struct {
	pipelines []*model.Pipeline
	resolver  *runner.TaskResolver
	options   *Options
	workDir   string
}

// New creates a new Agent with discovered skills.
func New(workDir string, opts *Options) (*Agent, error) {
	if opts == nil {
		opts = &Options{}
	}

	// Load skill pipelines from local .atkins/skills/
	loader := runner.NewSkillsLoader(workDir, workDir)
	pipelines, err := loader.Load()
	if err != nil {
		// Not fatal - may have no skills
		pipelines = []*model.Pipeline{}
	}

	// Merge global skills from $HOME/.atkins/skills/ (unless jailed)
	if !opts.Jail {
		if home, err := os.UserHomeDir(); err == nil {
			globalLoader := runner.NewSkillsLoader(workDir, workDir)
			globalLoader.SkillsDirs = []string{filepath.Join(home, ".atkins", "skills")}
			if globalPipelines, globalErr := globalLoader.Load(); globalErr == nil {
				seen := make(map[string]bool)
				for _, p := range pipelines {
					if p.ID != "" {
						seen[p.ID] = true
					}
				}
				for _, gp := range globalPipelines {
					if !seen[gp.ID] {
						pipelines = append(pipelines, gp)
					}
				}
			}
		}
	}

	// Try to load main pipeline config too
	if configPath, configDir, err := runner.DiscoverConfigFromCwd(); err == nil && configPath != "" {
		if mainPipelines, loadErr := runner.LoadPipeline(configPath); loadErr == nil {
			// Prepend main pipeline(s) - they take priority
			pipelines = append(mainPipelines, pipelines...)
			_ = configDir // Could chdir here if needed
		}
	}

	resolver := runner.NewTaskResolver(pipelines)

	return &Agent{
		pipelines: pipelines,
		resolver:  resolver,
		options:   opts,
		workDir:   workDir,
	}, nil
}

// Run starts the interactive REPL.
func (a *Agent) Run(ctx context.Context, version string) error {
	m := NewModel(a, version)
	p := tea.NewProgram(m,
		tea.WithContext(ctx),
	)

	_, err := p.Run()
	return err
}

// Pipelines returns the loaded pipelines.
func (a *Agent) Pipelines() []*model.Pipeline {
	return a.pipelines
}

// Resolver returns the task resolver.
func (a *Agent) Resolver() *runner.TaskResolver {
	return a.resolver
}

// Options returns the agent options.
func (a *Agent) Options() *Options {
	return a.options
}

// WorkDir returns the working directory.
func (a *Agent) WorkDir() string {
	return a.workDir
}

// Exec processes a single prompt non-interactively and exits.
func (a *Agent) Exec(ctx context.Context, prompt, version string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("empty prompt")
	}

	parser := NewParser(a.resolver, a.pipelines)
	greeter := NewGreeter()
	aliases := parser.Aliases()

	// Check alias
	if taskName := aliases.Match(prompt); taskName != "" {
		if resolved, err := a.resolver.Resolve(taskName); err == nil {
			return a.execTask(ctx, resolved)
		}
	}

	// Parse intent
	intent, err := parser.Parse(prompt)
	if err != nil {
		return err
	}

	switch intent.Type {
	case IntentTask:
		if intent.Resolved == nil {
			return fmt.Errorf("could not resolve: %s", prompt)
		}
		return a.execTask(ctx, intent.Resolved)

	case IntentHelp:
		fmt.Println("Usage: atkins -x \"<prompt>\"")
		fmt.Println("  Run a task, shell command, or ask for a greeting/fortune.")
		return nil

	case IntentSlash:
		return fmt.Errorf("slash commands are only available in interactive mode")

	case IntentQuit:
		return nil

	default:
		// Greeting
		if response := greeter.Match(prompt); response != "" {
			fmt.Println(response)
			return nil
		}

		// Fortune
		if MatchFortune(prompt) {
			fmt.Println(Fortune())
			return nil
		}

		// Shell fallback
		fields := strings.Fields(prompt)
		if len(fields) > 0 {
			if _, err := exec.LookPath(fields[0]); err == nil {
				return a.execShell(ctx, prompt)
			}
		}

		// Shell history single match
		shellHistory := NewShellHistory()
		if histMatches := shellHistory.Match(prompt); len(histMatches) == 1 {
			cmd := histMatches[0].Command
			if hFields := strings.Fields(cmd); len(hFields) > 0 {
				if _, err := exec.LookPath(hFields[0]); err == nil {
					return a.execShell(ctx, cmd)
				}
			}
		}

		return fmt.Errorf("unknown command: %s", prompt)
	}
}

func (a *Agent) execTask(ctx context.Context, task *model.ResolvedTask) error {
	start := time.Now()
	err := runner.RunPipeline(ctx, task.Pipeline, runner.PipelineOptions{
		Jobs:         []string{task.Job.Name},
		Silent:       true,
		Debug:        a.options.Debug,
		AllPipelines: a.pipelines,
	})
	dur := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s %s\n",
			colors.BrightRed("✗"), task.Name,
			colors.BrightRed("FAIL")+" "+colors.Dim(fmt.Sprintf("%.2fs", dur.Seconds())))
		return err
	}
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		colors.BrightGreen("✓"), task.Name,
		colors.BrightGreen("OK")+" "+colors.Dim(fmt.Sprintf("%.2fs", dur.Seconds())))
	return nil
}

func (a *Agent) execShell(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = a.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
