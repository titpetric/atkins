package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/titpetric/atkins-ci/colors"
	"github.com/titpetric/atkins-ci/model"
	"github.com/titpetric/atkins-ci/runner"
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
	var quiet bool
	var veryQuiet bool
	var verbose bool

	flag.StringVar(&pipelineFile, "file", "atkins.yml", "Path to pipeline file")
	flag.StringVar(&job, "job", "", "Specific job to run (optional, runs default if empty)")
	flag.BoolVar(&quiet, "q", false, "Quiet mode (suppress stdout from executed statements)")
	flag.BoolVar(&veryQuiet, "qq", false, "Very quiet mode (suppress stdout and stderr from executed statements)")
	flag.BoolVar(&verbose, "v", false, "Verbose mode (print command output)")
	flag.Parse()

	// Determine quiet mode level
	quietMode := 0
	if veryQuiet {
		quietMode = 2
	} else if quiet {
		quietMode = 1
	} else if !verbose {
		// Default: buffer output, don't print it
		quietMode = 1
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

	var wg sync.WaitGroup
	wg.Add(len(pipelines))
	var exitCode int
	var failedPipeline string
	for _, pipeline := range pipelines {
		if err := runPipeline(&wg, pipeline, job, quietMode); err != nil {
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

func runPipeline(wg *sync.WaitGroup, pipeline *model.Pipeline, job string, quietMode int) error {
	defer wg.Done()

	// Create execution tree
	tree := runner.NewExecutionTree(pipeline.Name)

	// Create execution context
	// Create tree renderer for in-place updates
	renderer := runner.NewTreeRenderer()

	ctx := &model.ExecutionContext{
		Variables: make(map[string]interface{}),
		Env:       make(map[string]string),
		Results:   make(map[string]interface{}),
		QuietMode: quietMode,
		Pipeline:  pipeline.Name,
		Depth:     0,
		Tree:      tree,
		Renderer:  renderer,
	}

	// Copy environment variables
	for _, env := range os.Environ() {
		k, v := parseEnv(env)
		if k != "" {
			ctx.Env[k] = v
		}
	}

	// Initial tree render
	renderer.Render(tree)

	// Execute jobs
	var jobsToRun map[string]*model.Job
	if job != "" {
		j, ok := pipeline.Jobs[job]
		if !ok {
			fatalf("%s Job '%s' not found\n", colors.BrightRed("ERROR:"), job)
		}
		jobsToRun = map[string]*model.Job{job: j}
	} else {
		jobsToRun = pipeline.Jobs
		if len(jobsToRun) == 0 {
			jobsToRun = pipeline.Tasks
		}
	}

	// Pre-populate all jobs as pending
	jobNodes := make(map[string]*runner.TreeNode)
	for jobName, jobDef := range jobsToRun {
		jobLabel := jobName
		if jobDef.Desc != "" {
			jobLabel = jobName + " - " + jobDef.Desc
		}
		jobNode := tree.AddJob(jobLabel)
		jobNodes[jobName] = jobNode
	}
	renderer.Render(tree)

	executor := runner.NewExecutor()
	for jobName, jobDef := range jobsToRun {
		jobCtx := *ctx
		jobCtx.Job = jobName
		jobCtx.JobDesc = jobDef.Desc
		jobCtx.Depth = 1

		// Get pre-created job node and mark it as running
		jobNode := jobNodes[jobName]
		jobNode.SetStatus(runner.StatusRunning)
		jobCtx.CurrentJob = jobNode
		renderer.Render(tree)

		// Pending jobs are shown in gray in the pre-rendered tree above

		if err := executor.ExecuteJob(&jobCtx, jobName, jobDef); err != nil {
			// Mark pipeline as failed
			tree.Root.Status = runner.StatusFailed
			// Render final tree
			renderer.Render(tree)
			fmt.Println(colors.BrightRed("✗ FAIL"))
			// Print stderr if there's any error output
			if runner.ErrorLog.Len() > 0 {
				fmt.Println(colors.BrightRed("Error output:"))
				fmt.Print(runner.ErrorLog.String())
			}
			return err
		}

		// Mark job as passed
		jobNode.SetStatus(runner.StatusPassed)

		// Render tree after job completes
		renderer.Render(tree)

		// Update parent context with step counts
		ctx.StepsCount += jobCtx.StepsCount
		ctx.StepsPassed += jobCtx.StepsPassed
	}

	// Mark pipeline as passed and render final tree
	tree.Root.Status = runner.StatusPassed
	renderer.Render(tree)
	fmt.Print(colors.BrightGreen(fmt.Sprintf("✓ PASS (%d steps passing)\n", ctx.StepsPassed)))
	return nil
}

func breadcrumb(ctx *model.ExecutionContext) string {
	parts := []string{ctx.Pipeline}
	if ctx.Job != "" {
		parts = append(parts, ctx.Job)
	}
	if ctx.Step != "" {
		parts = append(parts, ctx.Step)
	}
	return strings.Join(parts, " > ")
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func parseEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return "", ""
}
