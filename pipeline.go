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

// loadSkillPipelines loads and merges skill pipelines from disk.
// Unconditionally loads all YAML files from .atkins/skills/ and $HOME/.atkins/skills/,
// then filters by Pipeline.When conditions.
func loadSkillPipelines(env *runner.Environment) ([]*model.Pipeline, error) {
	dirs := runner.SkillsDirs(env.Root)
	skillPipelines, err := runner.DiscoverSkillsFromDirs(dirs)
	if err != nil {
		return nil, err
	}

	if len(skillPipelines) == 0 {
		return nil, fmt.Errorf("no skill pipelines loaded")
	}

	return skillPipelines, nil
}

// stdinHasData checks if stdin has data available without blocking.
// Returns true if stdin is piped/redirected with data available.
func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is not a terminal (i.e., is piped or redirected)
	return (stat.Mode() & os.ModeCharDevice) == 0
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

		if fileExplicitlySet {
			// If -f/--file was explicitly provided, use it directly
			absPath, err = filepath.Abs(opts.File)
			if err != nil {
				return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
			}
		} else {
			// Discover config file by traversing parent directories
			configPath, configDir, discoverErr := runner.DiscoverConfigFromCwd()
			if discoverErr != nil {
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
				pipelines, err = loadSkillPipelines(env)
				if err != nil {
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
				}
				opts.File = "<autodiscovered>"
				goto pipelineReady
			}
			absPath = configPath
			opts.File = configPath

			// Change to the directory containing the config file
			if err := os.Chdir(configDir); err != nil {
				return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), configDir, err)
			}
		}

		// Load and parse pipeline
		pipelines, err = runner.LoadPipeline(absPath)
		if err != nil {
			return fmt.Errorf("%s %s", colors.BrightRed("ERROR:"), err)
		}

		// Merge autodiscovered skills into the loaded pipeline
		if env, envErr := runner.DiscoverEnvironmentFromCwd(); envErr == nil {
			if skillPipelines, skillErr := loadSkillPipelines(env); skillErr == nil {
				pipelines = append(pipelines, skillPipelines...)
			}
		}
	}

pipelineReady:

	// Handle working directory override (applies to both stdin and file modes)
	if opts.WorkingDirectory != "" {
		if err := os.Chdir(opts.WorkingDirectory); err != nil {
			return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), opts.WorkingDirectory, err)
		}
	}

	if len(pipelines) == 0 {
		return fmt.Errorf("%s No pipelines found", colors.BrightRed("ERROR:"))
	}

	// Handle lint mode
	if opts.Lint || opts.List {
		for _, pipeline := range pipelines {
			linter := runner.NewLinter(pipeline)
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
			fmt.Printf("%s Pipeline '%s' is valid\n", colors.BrightGreen("✓"), pipelines[0].Name)
			return nil
		}
	}

	// Handle list mode
	if opts.List {
		runner.ListPipelines(pipelines)

		if opts.Debug {
			for _, pipeline := range pipelines {
				b, _ := yaml.Marshal(pipeline)
				fmt.Printf("%s\n", string(b))
			}
		}

		return nil
	}

	// Run pipeline(s)
	var exitCode int
	var failedPipeline string

	for _, pipeline := range pipelines {
		err := runner.RunPipeline(ctx, pipeline, runner.PipelineOptions{
			Job:          opts.Job,
			LogFile:      opts.LogFile,
			PipelineFile: opts.File,
			Debug:        opts.Debug,
			FinalOnly:    opts.FinalOnly,
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
