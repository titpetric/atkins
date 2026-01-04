package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/titpetric/atkins-ci/colors"
	"github.com/titpetric/atkins-ci/runner"
	"gopkg.in/yaml.v2"
)

func fatalf(message string, args ...any) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}

func fatal(message string) {
	fatalf("%s", message)
}

func main() {
	var pipelineFile string
	var job string
	var listFlag bool
	var lintFlag bool
	var debug bool

	flag.StringVar(&pipelineFile, "file", "atkins.yml", "Path to pipeline file")
	flag.StringVar(&job, "job", "", "Specific job to run (optional)")
	flag.BoolVar(&listFlag, "l", false, "List pipeline jobs and dependencies")
	flag.BoolVar(&lintFlag, "lint", false, "Lint pipeline for errors")
	flag.BoolVar(&debug, "debug", false, "Print debug data")
	flag.Parse()

	// Handle positional argument as job name
	args := flag.Args()
	if len(args) > 0 {
		if args[0] == "lint" {
			lintFlag = true
		} else {
			job = args[0]
		}
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(pipelineFile)
	if err != nil {
		fatalf("%s %v\n", colors.BrightRed("ERROR:"), err)
	}

	// Load and parse pipeline
	pipelines, err := runner.LoadPipeline(absPath)
	if err != nil {
		fatalf("%s %s\n", colors.BrightRed("ERROR:"), err)
	}

	if len(pipelines) == 0 {
		fatalf("%s No pipelines found\n", colors.BrightRed("ERROR:"))
	}

	// Handle lint mode
	if lintFlag {
		for _, pipeline := range pipelines {
			linter := runner.NewLinter(pipeline)
			errors := linter.Lint()
			if len(errors) > 0 {
				fmt.Printf("%s Pipeline '%s' has errors:\n", colors.BrightRed("✗"), pipeline.Name)
				for _, lintErr := range errors {
					fmt.Printf("  %s: %s\n", lintErr.Job, lintErr.Detail)
				}
				os.Exit(1)
			}
		}
		fmt.Printf("%s Pipeline '%s' is valid\n", colors.BrightGreen("✓"), pipelines[0].Name)
		return
	}

	// Handle list mode
	if listFlag {
		for _, pipeline := range pipelines {
			linter := runner.NewLinter(pipeline)
			errors := linter.Lint()
			if len(errors) > 0 {
				fmt.Printf("%s Pipeline '%s' has dependency errors:\n", colors.BrightRed("✗"), pipeline.Name)
				for _, lintErr := range errors {
					fmt.Printf("  %s: %s\n", lintErr.Job, lintErr.Detail)
				}
				os.Exit(1)
			}

			if debug {
				b, _ := yaml.Marshal(pipeline)
				fmt.Printf("%s\n", string(b))
			}

			runner.ListPipeline(pipeline)
		}
		return
	}

	// Run pipeline(s)
	var wg sync.WaitGroup
	wg.Add(len(pipelines))
	var exitCode int
	var failedPipeline string

	ctx := context.TODO()

	for _, pipeline := range pipelines {
		if err := runner.RunPipeline(ctx, &wg, pipeline, job); err != nil {
			exitCode = 1
			failedPipeline = pipeline.Name
		}
	}
	wg.Wait()

	// Print any captured error output with formatting
	runner.ErrorLogMutex.Lock()
	if runner.ErrorLog.Len() > 0 {
		fmt.Fprintf(os.Stderr, "\nAn error occurred in %q pipeline:\n\n", failedPipeline)
		fmt.Fprintf(os.Stderr, "  Exit code: %d\n", runner.LastExitCode)
		fmt.Fprintf(os.Stderr, "  Error output:\n")
		// Indent the error output
		for _, line := range strings.Split(runner.ErrorLog.String(), "\n") {
			if line != "" {
				fmt.Fprintf(os.Stderr, "    %s\n", line)
			}
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
	runner.ErrorLogMutex.Unlock()

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
