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
	"github.com/titpetric/atkins/runner"
)

// Pipeline provides a cli.Command that runs the atkins command pipeline.
func Pipeline() *cli.Command {
	var pipelineFile string
	var job string
	var listFlag bool
	var lintFlag bool
	var debug bool
	var logFile string
	var finalOutputOnly bool
	var workingDirectory string
	var fileFlag *pflag.Flag

	return &cli.Command{
		Name:    "run",
		Title:   "Pipeline automation tool",
		Default: true,
		Bind: func(fs *pflag.FlagSet) {
			fs.StringVarP(&pipelineFile, "file", "f", "", "Path to pipeline file (auto-discovers .atkins.yml)")
			fs.StringVar(&job, "job", "", "Specific job to run")
			fs.BoolVarP(&listFlag, "list", "l", false, "List pipeline jobs and dependencies")
			fs.BoolVar(&lintFlag, "lint", false, "Lint pipeline for errors")
			fs.BoolVar(&debug, "debug", false, "Print debug data")
			fs.StringVar(&logFile, "log", "", "Log file path for command execution")
			fs.BoolVar(&finalOutputOnly, "final", false, "Only render final output without redrawing (no interactive tree)")
			fs.StringVarP(&workingDirectory, "working-directory", "w", "", "Change to this directory before running")
			fileFlag = fs.Lookup("file")
		},
		Run: func(ctx context.Context, args []string) error {
			// Handle working directory first, before anything else
			if workingDirectory != "" {
				if err := os.Chdir(workingDirectory); err != nil {
					return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), workingDirectory, err)
				}
			}

			// Track if file was explicitly provided
			fileExplicitlySet := fileFlag != nil && fileFlag.Changed

			// Handle positional arguments
			for _, arg := range args {
				// Check if arg is an existing regular file (shebang invocation)
				if info, err := os.Stat(arg); err == nil && info.Mode().IsRegular() {
					pipelineFile = arg
					fileExplicitlySet = true
					continue
				}

				if job == "" {
					// Treat as job name if not already set
					job = arg
				}
			}

			var absPath string
			var err error

			if fileExplicitlySet {
				// If -f/--file was explicitly provided, use it directly without changing workdir
				absPath, err = filepath.Abs(pipelineFile)
				if err != nil {
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
				}
			} else {
				// Discover config file by traversing parent directories
				configPath, configDir, err := runner.DiscoverConfigFromCwd()
				if err != nil {
					return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
				}
				absPath = configPath
				pipelineFile = configPath

				// Change to the directory containing the config file
				if err := os.Chdir(configDir); err != nil {
					return fmt.Errorf("%s failed to change directory to %s: %v", colors.BrightRed("ERROR:"), configDir, err)
				}
			}

			// Load and parse pipeline
			pipelines, err := runner.LoadPipeline(absPath)
			if err != nil {
				return fmt.Errorf("%s %s", colors.BrightRed("ERROR:"), err)
			}

			if len(pipelines) == 0 {
				return fmt.Errorf("%s No pipelines found", colors.BrightRed("ERROR:"))
			}

			// Handle lint mode
			if lintFlag || listFlag {
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
				fmt.Printf("%s Pipeline '%s' is valid\n", colors.BrightGreen("✓"), pipelines[0].Name)
				if lintFlag {
					return nil
				}
			}

			// Handle list mode
			if listFlag {
				for _, pipeline := range pipelines {
					if debug {
						b, _ := yaml.Marshal(pipeline)
						fmt.Printf("%s\n", string(b))
					}

					if err := runner.ListPipeline(pipeline); err != nil {
						return err
					}
				}
				return nil
			}

			// Run pipeline(s)
			var exitCode int
			var failedPipeline string

			for _, pipeline := range pipelines {
				err := runner.RunPipeline(ctx, pipeline, runner.PipelineOptions{
					Job:          job,
					LogFile:      logFile,
					PipelineFile: pipelineFile,
					Debug:        debug,
					FinalOnly:    finalOutputOnly,
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
		},
	}
}
