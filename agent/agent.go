package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	agentrouter "github.com/titpetric/atkins/agent/router"
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

	// Use centralized router and executor
	registry := DefaultRegistry()
	rtr := agentrouter.NewRouter(a.resolver, a.pipelines, registry)
	route := rtr.Route(prompt)

	out := NewStdOutput()
	exec := NewExecutor(ctx, a, rtr, out)
	return exec.ExecuteRoute(route)
}
