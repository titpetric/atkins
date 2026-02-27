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

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// loadSkillPipelines loads skill pipelines from the project-local .atkins/skills/ directory.
// Global skills from $HOME/.atkins/skills/ are loaded separately with correct working directory.
// workDir is used for when.files checks; if empty, uses current working directory.
func loadSkillPipelines(projectRoot string, workDir string, opts *Options) ([]*model.Pipeline, error) {
	skills := runner.NewSkills(projectRoot, true)
	skills.WorkDir = workDir
	return skills.Load()
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

// resolveJobTarget determines which pipeline and job to run based on the job name.
// Resolution order:
// 0. Explicit root reference (e.g., ":build" or ":go:build") - bypass alias resolution
// 1. Prefixed job (e.g., "go:test") - explicit skill:job reference
// 2. Exact main pipeline match - job name exactly matches a job in main pipeline
// 3. Alias match - job with matching alias in any skill pipeline
// 4. Skill ID with default - skill name that has a "default" job
// 5. Skill ID (for listing) - skill name without requiring default job
// 6. Fuzzy match - suffix/substring match in job names (if exactly one match)
// 7. Main pipeline - fallback to first pipeline (no skill ID)
func resolveJobTarget(pipelines []*model.Pipeline, jobName string) ([]*model.Pipeline, string, error) {
	// 0. Check for explicit root reference (leading colon)
	// :build → main pipeline job "build"
	// :go:build → skill "go" job "build" (explicit, bypasses aliases)
	if strings.HasPrefix(jobName, ":") {
		explicitName := jobName[1:] // Remove leading colon

		// Check if it's :skillID:jobName or just :jobName
		if parts := strings.SplitN(explicitName, ":", 2); len(parts) == 2 {
			// :go:build → skill "go", job "build"
			skillID, skillJob := parts[0], parts[1]
			for _, p := range pipelines {
				if p.ID == skillID {
					return []*model.Pipeline{p}, skillJob, nil
				}
			}
			return nil, "", fmt.Errorf("%s skill %q not found", colors.BrightRed("ERROR:"), skillID)
		}

		// :build → main pipeline (ID="") job "build"
		for _, p := range pipelines {
			if p.ID == "" {
				return []*model.Pipeline{p}, explicitName, nil
			}
		}
		return nil, "", fmt.Errorf("%s main pipeline not found", colors.BrightRed("ERROR:"))
	}

	// 1. Check if job has a skill prefix (e.g., "go:test")
	if parts := strings.SplitN(jobName, ":", 2); len(parts) == 2 {
		skillID, skillJob := parts[0], parts[1]
		for _, p := range pipelines {
			if p.ID == skillID {
				return []*model.Pipeline{p}, skillJob, nil
			}
		}
		return nil, "", fmt.Errorf("%s skill %q not found", colors.BrightRed("ERROR:"), skillID)
	}

	// 2. Check if jobName exactly matches a job in the main pipeline
	// Main pipeline jobs take precedence over aliases
	for _, p := range pipelines {
		if p.ID == "" {
			jobs := p.Jobs
			if len(jobs) == 0 {
				jobs = p.Tasks
			}
			if _, exists := jobs[jobName]; exists {
				return []*model.Pipeline{p}, jobName, nil
			}
			break // Only check the main pipeline (ID="")
		}
	}

	// 3. Check if jobName matches an alias in any pipeline
	for _, p := range pipelines {
		jobs := p.Jobs
		if len(jobs) == 0 {
			jobs = p.Tasks
		}
		for jn, job := range jobs {
			for _, alias := range job.Aliases {
				if alias == jobName {
					return []*model.Pipeline{p}, jn, nil
				}
			}
		}
	}

	// 4. Check if jobName matches a skill ID with a "default" job
	for _, p := range pipelines {
		if p.ID == jobName {
			jobs := p.Jobs
			if len(jobs) == 0 {
				jobs = p.Tasks
			}
			if _, hasDefault := jobs["default"]; hasDefault {
				return []*model.Pipeline{p}, "default", nil
			}
		}
	}

	// 5. Check if jobName matches a skill ID (for listing without default)
	for _, p := range pipelines {
		if p.ID == jobName {
			return []*model.Pipeline{p}, "", nil
		}
	}

	// 6. Fuzzy match - check for suffix/substring matches in job names
	matches := findFuzzyMatches(pipelines, jobName)
	if len(matches) == 1 {
		// Exactly one match found, use it
		match := matches[0]
		return []*model.Pipeline{match.Pipeline}, match.JobName, nil
	} else if len(matches) > 1 {
		// Multiple matches found, list them and exit
		return []*model.Pipeline{matches[0].Pipeline}, "", &FuzzyMatchError{Matches: matches}
	}

	// 7. Fallback to main pipeline (first one, no skill ID)
	return []*model.Pipeline{pipelines[0]}, jobName, nil
}

func runPipeline(ctx context.Context, opts *Options, args []string) error {
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

		if opts.Job == "" {
			// Treat as job name if not already set
			opts.Job = arg
		}
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
			if env, envErr := runner.DiscoverEnvironmentFromCwd(); envErr == nil {
				if skillPipelines, skillErr := loadSkillPipelines(env.Root, originalCwd, opts); skillErr == nil {
					pipelines = append(pipelines, skillPipelines...)
				}
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
	// Uses originalCwd for when.files checks since cwd may have changed.
	if !opts.Jail {
		globalSkills := runner.NewGlobalSkills()
		globalSkills.WorkDir = originalCwd
		if globalPipelines, globalErr := globalSkills.Load(); globalErr == nil {
			seen := make(map[string]bool)
			for _, p := range pipelines {
				if p.ID != "" {
					seen[p.ID] = true
				}
			}
			for _, gp := range globalPipelines {
				if !seen[gp.ID] {
					// Set Dir so global skills execute from the original cwd,
					// not from the config directory that atkins may have changed to.
					if gp.Dir == "" {
						gp.Dir = originalCwd
					}
					pipelines = append(pipelines, gp)
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

	// Determine which pipeline(s) to target based on job specification
	jobName := opts.Job
	if jobName != "" {
		var err error
		pipelines, jobName, err = resolveJobTarget(pipelines, jobName)
		if err != nil {
			// Check if it's a fuzzy match error with multiple matches
			var fuzzyErr *FuzzyMatchError
			if errors.As(err, &fuzzyErr) {
				fmt.Fprintf(os.Stderr, "%s found %d matching jobs:\n\n", colors.BrightYellow("INFO:"), len(fuzzyErr.Matches))
				for _, match := range fuzzyErr.Matches {
					fmt.Fprintf(os.Stderr, "  - %s\n", colors.BrightOrange(match.FullName))
				}
				os.Exit(1)
			}
			return err
		}
	}

	// Handle list mode
	if opts.List {
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

	// When no job is specified, only run the main pipeline (no ID)
	// Skills (pipelines with IDs) are never run automatically
	if jobName == "" {
		var main *model.Pipeline
		for _, p := range pipelines {
			if p.ID == "" {
				main = p
				break
			}
		}
		if main != nil {
			pipelines = []*model.Pipeline{main}
		}
	}

	// Run pipeline(s)
	var exitCode int
	var failedPipeline string

	for _, pipeline := range pipelines {
		err := runner.RunPipeline(ctx, pipeline, runner.PipelineOptions{
			Job:          jobName,
			LogFile:      opts.LogFile,
			PipelineFile: opts.File,
			Debug:        opts.Debug,
			FinalOnly:    opts.FinalOnly,
			JSON:         opts.JSON,
			YAML:         opts.YAML,
			AllPipelines: allPipelines,
		})
		if err != nil {
			exitCode = 1
			failedPipeline = pipeline.Name

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
