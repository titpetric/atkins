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

	// Use centralized router (follows structure.d2 flow)
	registry := DefaultRegistry()
	router := NewRouter(a.resolver, a.pipelines, registry)
	route := router.Route(prompt)

	switch route.Type {
	case RouteTask, RouteAlias:
		if route.Resolved == nil {
			return fmt.Errorf("could not resolve: %s", prompt)
		}
		return a.execTask(ctx, route.Resolved)

	case RouteMultiTask:
		// Run multiple tasks in sequence
		for _, task := range route.Tasks {
			if err := a.execTask(ctx, task); err != nil {
				return err // Stop on first failure
			}
		}
		return nil

	case RouteConfirm:
		// In non-interactive mode, show suggestion and fail
		fmt.Printf("Did you mean %s?\n", route.Suggestion)
		fmt.Printf("Run: atkins -x \"%s\"\n", route.Suggestion)
		return fmt.Errorf("unknown command: %s", route.Original)

	case RouteHelp:
		fmt.Print(UsageText())
		return nil

	case RouteSlash:
		// Handle some slash commands in non-interactive mode
		switch route.Command {
		case "list", "l", "ls", "skills":
			a.printSkillList()
			return nil
		case "help", "h", "?":
			fmt.Print(UsageText())
			return nil
		case "aliases", "alias":
			a.printAliases(router.Aliases())
			return nil
		default:
			return fmt.Errorf("slash command /%s is only available in interactive mode", route.Command)
		}

	case RouteQuit:
		return nil

	case RouteGreeting:
		fmt.Println(route.Greeting)
		return nil

	case RouteFortune:
		fmt.Println(route.Fortune)
		return nil

	case RouteCorrection:
		router.Aliases().Add(route.Phrase, route.AliasTask)
		fmt.Printf("Got it! \"%s\" will now run %s\n", route.Phrase, route.AliasTask)
		return nil

	case RouteShell:
		return a.execShell(ctx, route.ShellCmd)

	default:
		// RouteUnknown
		if route.Ambiguous && len(route.Matches) > 0 {
			fmt.Println("Matching skills:")
			for _, match := range route.Matches {
				fmt.Println("  " + match)
			}
			fmt.Println("\nBe more specific or use the full skill name")
			return nil
		}
		return fmt.Errorf("unknown command: %s", prompt)
	}
}

// printSkillList prints available skills for non-interactive mode.
func (a *Agent) printSkillList() {
	if len(a.pipelines) == 0 {
		fmt.Println("No skills available")
		return
	}

	fmt.Println("Available skills:")
	for _, p := range a.pipelines {
		var prefix string
		if p.ID != "" {
			prefix = p.ID + ":"
		}

		for name, job := range p.Jobs {
			fullName := prefix + name
			if job.Desc != "" {
				fmt.Printf("  %s - %s\n", fullName, job.Desc)
			} else {
				fmt.Printf("  %s\n", fullName)
			}
		}
	}
}

// printAliases prints defined aliases for non-interactive mode.
func (a *Agent) printAliases(aliases *AliasStore) {
	if len(aliases.Aliases) == 0 {
		fmt.Println("No aliases defined.")
		fmt.Println("\nTeach an alias with:")
		fmt.Println("  alias <phrase> to <command>")
		return
	}

	fmt.Println("Defined aliases:")
	for _, alias := range aliases.Aliases {
		fmt.Printf("  %s as %s\n", alias.Phrase, alias.Prompt)
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
