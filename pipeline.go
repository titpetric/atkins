package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/titpetric/cli"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/agent"
	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
	"github.com/titpetric/atkins/version"
)

// loadSkillPipelines loads skill pipelines from the project-local .atkins/skills/ directory.
// workspaceDir is the folder containing .atkins/ (used as Dir for skills without when:).
// startDir is where to start searching for when: files (typically user's cwd).
func loadSkillPipelines(workspaceDir string, startDir string, opts *Options) ([]*model.Pipeline, error) {
	loader := runner.NewSkillsLoader(workspaceDir, startDir)
	return loader.Load()
}

// stdinHasData checks if stdin has data available without blocking.
// Returns true if stdin is piped/redirected with data available.
func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is not a terminal (i.e., is piped or redirected)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return false
	}
	// For regular files, check size. For pipes, size is 0 but data may be available.
	// Only treat as having data if it's a regular file with content, or a named pipe.
	// This avoids blocking on empty pipes (e.g., from `timeout` command).
	mode := stat.Mode()
	if mode.IsRegular() {
		return stat.Size() > 0
	}
	// For named pipes (FIFOs), assume data is available
	if mode&os.ModeNamedPipe != 0 {
		return true
	}
	// For anonymous pipes, we can't easily check without blocking.
	// Check if it looks like a pipe from a shell command (mode is irregular)
	// by checking if we're NOT a socket, device, etc.
	if mode&(os.ModeSocket|os.ModeDevice) == 0 {
		return true
	}
	return false
}

// Pipeline provides a cli.Command that runs the atkins command pipeline.
func Pipeline() *cli.Command {
	opts := NewOptions()

	return &cli.Command{
		Name:    "run",
		Title:   "Pipeline automation tool",
		Default: true,
		Bind: func(fs *pflag.FlagSet) {
			opts.Bind(fs)
		},
		Run: func(ctx context.Context, args []string) error {
			return runPipeline(ctx, opts, args)
		},
	}
}

func runPipeline(ctx context.Context, opts *Options, args []string) error {
	// Handle version flag early, before any file discovery
	if opts.Version {
		return version.Run(version.Info{
			Version:    Version,
			Commit:     Commit,
			CommitTime: CommitTime,
			Branch:     Branch,
		})
	}

	// Handle agent mode
	if opts.Agent {
		return runAgent(ctx, opts)
	}

	// Handle exec mode (-x "prompt")
	if opts.Exec != "" {
		return runExec(ctx, opts)
	}

	// Validate mutually exclusive flags
	if opts.JSON && opts.YAML {
		return fmt.Errorf("%s --json and --yaml flags cannot be combined", colors.BrightRed("ERROR:"))
	}

	fileFlag := opts.FlagSet.Lookup("file")

	// Handle positional arguments before changing directory
	fileExplicitlySet := fileFlag != nil && fileFlag.Changed
	for _, arg := range args {
		// Check if arg is an existing regular file (shebang invocation)
		if info, err := os.Stat(arg); err == nil && info.Mode().IsRegular() {
			opts.File = arg
			fileExplicitlySet = true
			continue
		}

		// Collect all job names from positional arguments
		opts.Jobs = append(opts.Jobs, arg)
	}

	// Save original working directory for global skill when.files checks,
	// since cwd may change during config/environment discovery.
	originalCwd, _ := os.Getwd()

	// Check stdin first (before file discovery)
	var pipelines []*model.Pipeline
	var err error

	if stdinHasData() {
		// Read pipeline from stdin
		pipelines, err = runner.LoadPipelineFromReader(os.Stdin)
		if err != nil {
			return fmt.Errorf("%s %s", colors.BrightRed("ERROR:"), err)
		}
		// Set default name if not specified
		if pipelines[0].Name == "" {
			pipelines[0].Name = "stdin"
		}
		opts.File = "stdin"
	} else {
		// Discover or resolve pipeline file before changing directory
		var absPath string
		var configDir string

		if fileExplicitlySet {
			// If -f/--file was explicitly provided, use it directly
			absPath, err = filepath.Abs(opts.File)
			if err != nil {
				return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
			}
		} else {
			// Discover config file by traversing parent directories
			var configPath string
			var discoverErr error
			configPath, configDir, discoverErr = runner.DiscoverConfigFromCwd()
			if discoverErr != nil && configPath != "" {
				// No config file found — try environment autodiscovery
				env, envErr := runner.DiscoverEnvironmentFromCwd()
				if envErr != nil {
					// Neither config nor environment found
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), discoverErr)
				}

				// Change to the discovered project root
				if err := os.Chdir(env.Root); err != nil {
					return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), env.Root, err)
				}

				// Load and merge skill pipelines
				pipelines, err = loadSkillPipelines(env.Root, originalCwd, opts)
				if err != nil {
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
				}
				opts.File = "<autodiscovered>"
				goto pipelineReady
			}
			absPath = configPath
			opts.File = configPath

			// Only change directory when a config file is found (not just .atkins/ folder).
			// For skills-only mode, stay in user's working directory.
			if configPath != "" {
				if err := os.Chdir(configDir); err != nil {
					return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), configDir, err)
				}
			}
		}

		// Load and parse pipeline from detected config path.
		// No config is given if an .atkins folder exists here.
		if absPath != "" {
			pipelines, err = runner.LoadPipeline(absPath)
			if err != nil {
				return fmt.Errorf("%s %s", colors.BrightRed("ERROR:"), err)
			}

			// Merge autodiscovered skills into the loaded pipeline
			if skillPipelines, skillErr := loadSkillPipelines(configDir, originalCwd, opts); skillErr == nil {
				pipelines = append(pipelines, skillPipelines...)
			}
		} else {
			// .atkins/ folder detected without config file - load skills as primary pipelines
			opts.File = ".atkins/"
			if skillPipelines, skillErr := loadSkillPipelines(configDir, originalCwd, opts); skillErr == nil {
				pipelines = skillPipelines
			}
		}
	}

pipelineReady:

	// Always merge global skills from $HOME/.atkins/skills/ (unless jailed).
	// Local .atkins/skills/ takes precedence: skip globals already loaded by ID.
	if !opts.Jail {
		if home, err := os.UserHomeDir(); err == nil {
			globalLoader := runner.NewSkillsLoader(originalCwd, originalCwd)
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

	// Handle working directory override (applies to both stdin and file modes)
	if opts.WorkingDirectory != "" {
		if err := os.Chdir(opts.WorkingDirectory); err != nil {
			return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), opts.WorkingDirectory, err)
		}
	}

	// Handle lint mode
	if opts.Lint || opts.List {
		for _, pipeline := range pipelines {
			linter := runner.NewLinterWithPipelines(pipeline, pipelines)
			lintErrors := linter.Lint()
			if len(lintErrors) > 0 {
				fmt.Printf("%s Pipeline '%s' has errors:\n", colors.BrightRed("✗"), pipeline.Name)
				for _, lintErr := range lintErrors {
					fmt.Printf("  %s: %s\n", lintErr.Job, lintErr.Detail)
				}
				return io.EOF
			}
		}
		if opts.Lint {
			if len(pipelines) > 0 {
				fmt.Printf("%s Pipeline '%s' is valid\n", colors.BrightGreen("✓"), pipelines[0].Name)
			}
			return nil
		}
	}

	// Save all pipelines for cross-pipeline task references
	allPipelines := pipelines

	// Handle list mode
	if opts.List {
		// If a job/skill name is specified, filter to that pipeline
		if len(opts.Jobs) > 0 {
			for _, p := range pipelines {
				if p.ID == opts.Jobs[0] {
					pipelines = []*model.Pipeline{p}
					break
				}
			}
		}

		if opts.JSON {
			return runner.ListPipelinesJSON(pipelines)
		}
		if opts.YAML {
			return runner.ListPipelinesYAML(pipelines)
		}

		runner.ListPipelines(pipelines)

		if opts.Debug {
			for _, pipeline := range pipelines {
				b, _ := yaml.Marshal(pipeline)
				fmt.Printf("%s\n", string(b))
			}
		}

		return nil
	}

	// When no jobs specified, run the default job from main pipeline
	if len(opts.Jobs) == 0 {
		opts.Jobs = []string{"default"}
	}

	// Resolve all jobs and group by pipeline
	type pipelineJobs struct {
		pipeline *model.Pipeline
		jobs     []string
	}
	pipelineJobsMap := make(map[*model.Pipeline]*pipelineJobs)
	var pipelineOrder []*model.Pipeline
	resolver := runner.NewTaskResolver(pipelines)

	// For implicit "default", resolve against the primary pipeline only
	// to avoid matching skill defaults when the main pipeline has none.
	if len(pipelines) > 0 {
		primaryResolver := runner.NewTaskResolver(pipelines[:1])
		if target, err := primaryResolver.Resolve("default"); err == nil {
			_ = target // primary has default, resolver will find it
		} else if len(opts.Jobs) == 1 && opts.Jobs[0] == "default" {
			resolver = primaryResolver
		}
	}

	for _, jobName := range opts.Jobs {
		target, err := resolver.Resolve(jobName)
		if err != nil {
			// Check if it's a fuzzy match error with multiple matches
			var fuzzyErr *runner.FuzzyMatchError
			if errors.As(err, &fuzzyErr) {
				fmt.Fprintf(os.Stderr, "%s found %d matching jobs:\n\n", colors.BrightYellow("INFO:"), len(fuzzyErr.Matches))
				for _, match := range fuzzyErr.Matches {
					displayName := match.Name
					if match.Pipeline.ID != "" {
						displayName = match.Pipeline.ID + ":" + match.Name
					}
					fmt.Fprintf(os.Stderr, "  - %s\n", colors.BrightOrange(displayName))
				}
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "%s %v\n", colors.BrightRed("ERROR:"), err)
			fmt.Fprintf(os.Stderr, "\nUsage: atkins [flags] [job-names...]\n")
			os.Exit(1)
		}

		pipeline := target.Pipeline
		if pipelineJobsMap[pipeline] == nil {
			pipelineJobsMap[pipeline] = &pipelineJobs{pipeline: pipeline}
			pipelineOrder = append(pipelineOrder, pipeline)
		}
		// Strip pipeline ID prefix since RunPipeline uses raw job map keys
		resolvedName := target.Name
		if pipeline.ID != "" {
			resolvedName = strings.TrimPrefix(resolvedName, pipeline.ID+":")
		}
		pipelineJobsMap[pipeline].jobs = append(pipelineJobsMap[pipeline].jobs, resolvedName)
	}

	// Run each pipeline with its collected jobs
	for _, pipeline := range pipelineOrder {
		pj := pipelineJobsMap[pipeline]
		err := runner.RunPipeline(ctx, pipeline, runner.PipelineOptions{
			Jobs:         pj.jobs,
			LogFile:      opts.LogFile,
			PipelineFile: opts.File,
			Debug:        opts.Debug,
			FinalOnly:    opts.FinalOnly,
			JSON:         opts.JSON,
			YAML:         opts.YAML,
			AllPipelines: allPipelines,
		})
		if err != nil {
			exitCode := 1
			failedPipeline := pipeline.Name

			var errorLog runner.ExecError
			if errors.As(err, &errorLog) {
				if errorLog.Len() > 0 {
					fmt.Fprintf(os.Stderr, "\nAn error occurred in %q pipeline:\n\n", failedPipeline)
					fmt.Fprintf(os.Stderr, "  Exit code: %d\n", errorLog.LastExitCode)
					fmt.Fprintf(os.Stderr, "  Error output:\n")
					for _, line := range strings.Split(errorLog.Output, "\n") {
						if line != "" {
							fmt.Fprintf(os.Stderr, "    %s\n", line)
						}
					}
				}
				exitCode = errorLog.LastExitCode
			} else {
				fmt.Fprintf(os.Stderr, "\nAn error occurred in %q pipeline:\n", failedPipeline)
				fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
			}

			if exitCode != 0 {
				os.Exit(exitCode)
			}
		}
	}
	return nil
}

// runAgent starts the interactive agent REPL.
func runAgent(ctx context.Context, opts *Options) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	agentOpts := &agent.Options{
		Debug:   opts.Debug,
		Verbose: false,
		Jail:    opts.Jail,
	}

	a, err := agent.New(cwd, agentOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	return a.Run(ctx, Version)
}

// runExec handles non-interactive prompt execution (-x flag).
func runExec(ctx context.Context, opts *Options) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	agentOpts := &agent.Options{
		Debug:   opts.Debug,
		Verbose: false,
		Jail:    opts.Jail,
	}

	a, err := agent.New(cwd, agentOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	return a.Exec(ctx, opts.Exec, Version)
}
