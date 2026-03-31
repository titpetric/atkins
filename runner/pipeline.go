package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	runnererrors "github.com/titpetric/atkins/runner/errors"
	"github.com/titpetric/atkins/treeview"
)

// PipelineOptions contains options for running a pipeline.
type PipelineOptions struct {
	Jobs         []string // Jobs to run (in order)
	LogFile      string
	PipelineFile string
	Debug        bool
	FinalOnly    bool
	JSON         bool
	YAML         bool
	AllPipelines []*model.Pipeline // All loaded pipelines for cross-pipeline task references
}

// Pipeline holds pipeline execution logic.
type Pipeline struct {
	opts PipelineOptions
	data *model.Pipeline
}

// NewPipeline allocates a new *Pipeline with dependencies.
func NewPipeline(data *model.Pipeline, opts PipelineOptions) *Pipeline {
	return &Pipeline{
		data: data,
		opts: opts,
	}
}

// buildAndAddStepsToJob adds step nodes to a job node with command children for multi-command steps
func buildAndAddStepsToJob(jobNode *treeview.TreeNode, steps []*model.Step) {
	for _, step := range steps {
		stepNode := treeview.NewPendingStepNode(step.DisplayLabel(), step.IsDeferred(), step.Summarize)
		stepNode.SetQuiet(step.Quiet)
		// Only add command child nodes if step has multiple commands (single command already shown in label)
		commands := step.Commands()
		if len(commands) > 1 {
			for _, cmd := range commands {
				stepNode.AddChild(treeview.NewCmdNode(cmd))
			}
		}
		jobNode.AddChild(stepNode)
	}
}

// RunPipeline runs a pipeline with the given options.
func RunPipeline(ctx context.Context, pipeline *model.Pipeline, opts PipelineOptions) error {
	var logger *eventlog.Logger
	if opts.LogFile != "" || opts.PipelineFile != "" {
		logger = eventlog.NewLogger(opts.LogFile, pipeline.Name, opts.PipelineFile, opts.Debug)
	}

	service := NewPipeline(pipeline, opts)

	return service.runPipeline(ctx, logger)
}

func (p *Pipeline) runPipeline(ctx context.Context, logger *eventlog.Logger) error {
	var (
		pipeline     = p.data
		jobs         = p.opts.Jobs
		finalOnly    = p.opts.FinalOnly
		outputJSON   = p.opts.JSON
		outputYAML   = p.opts.YAML
		silentOutput = outputJSON || outputYAML
	)

	tree := treeview.NewBuilder(pipeline.Name)
	root := tree.Root()

	var display *treeview.Display
	if silentOutput {
		display = treeview.NewSilentDisplay()
	} else {
		display = treeview.NewDisplayWithFinal(finalOnly)
	}

	pipelineCtx := &ExecutionContext{
		Variables:    NewContextVariables(nil),
		Env:          make(map[string]string),
		Results:      make(map[string]any),
		Pipeline:     pipeline,
		AllPipelines: p.opts.AllPipelines,
		Depth:        0,
		Builder:      tree,
		Display:      display,
		Context:      ctx,
		JobNodes:     make(map[string]*treeview.TreeNode),
		EventLogger:  logger,
		jobTracker:   newJobTracker(),
	}

	// Copy environment variables from OS
	for _, env := range os.Environ() {
		k, v := parseEnv(env)
		if k != "" {
			pipelineCtx.Env[k] = v
		}
	}

	// Evaluate pipeline-level working directory BEFORE merging variables,
	// so that $(command) interpolation in vars runs from the correct directory.
	if pipeline.Dir != "" {
		dir, err := InterpolateString(pipeline.Dir, pipelineCtx)
		if err != nil {
			return fmt.Errorf("failed to interpolate pipeline dir %q: %w", pipeline.Dir, err)
		}
		if info, statErr := os.Stat(dir); statErr != nil {
			return fmt.Errorf("pipeline dir %q: %w", dir, statErr)
		} else if !info.IsDir() {
			return fmt.Errorf("pipeline dir %q is not a directory", dir)
		}
		pipelineCtx.Dir = dir
	}

	if err := MergeVariables(pipelineCtx, pipeline.Decl); err != nil {
		return err
	}

	// Resolve jobs to run
	allJobs := pipeline.GetJobs()

	// Resolve dependencies for all requested jobs, collecting into unified order
	var jobOrder []string
	seenJobs := make(map[string]bool)
	for _, job := range jobs {
		order, err := ResolveJobDependencies(allJobs, job)
		if err != nil {
			var noDefaultErr *runnererrors.NoDefaultJobError
			if errors.As(err, &noDefaultErr) {
				// Print available jobs and error message similar to task
				fmt.Fprintf(os.Stderr, "%s Available jobs for this project:\n", colors.BrightYellow("atkins:"))
				printAvailableJobs(noDefaultErr.Jobs, pipeline.ID)
				fmt.Fprintf(os.Stderr, "%s Job %q does not exist\n", colors.BrightRed("atkins:"), "default")
				os.Exit(1)
			}
			fmt.Printf("%s %s\n", colors.BrightRed("ERROR:"), err)
			os.Exit(1)
		}
		// Add jobs to unified order, skipping duplicates
		for _, name := range order {
			if !seenJobs[name] {
				seenJobs[name] = true
				jobOrder = append(jobOrder, name)
			}
		}
	}

	// Pre-populate all jobs as pending - include all jobs that might be invoked
	jobNodes := make(map[string]*treeview.TreeNode)
	jobsToCreate := make(map[string]bool)

	// Track cross-pipeline jobs separately (key is "skillID:jobName")
	crossPipelineJobs := make(map[string]*model.Job)

	// Task resolvers: skill-local and global
	globalResolver := NewTaskResolver(p.opts.AllPipelines)
	skillResolver := NewSkillResolver(pipeline)

	// Helper to resolve a task name to its job and canonical name.
	resolveTaskName := func(taskName string) (string, *model.Job, error) {
		resolved, err := skillResolver.ResolveWithFallback(taskName, globalResolver)
		if err != nil {
			return "", nil, err
		}

		canonicalName := resolved.Name

		// Track cross-pipeline jobs for later lookup
		if resolved.Pipeline != pipeline {
			crossPipelineJobs[canonicalName] = resolved.Job
		} else if pipeline.ID != "" {
			// For same-pipeline jobs, strip the prefix to match allJobs keys.
			// This ensures consistent naming: allJobs has non-prefixed keys,
			// and jobNodes should use the same keys for lookup during execution.
			canonicalName = strings.TrimPrefix(canonicalName, pipeline.ID+":")
		}

		return canonicalName, resolved.Job, nil
	}

	// Recursively find all jobs that might be invoked
	var findInvokedJobs func(taskRef string, parentJobName string) error
	findInvokedJobs = func(taskRef string, parentJobName string) error {
		// Resolve the task reference
		canonicalName, job, err := resolveTaskName(taskRef)
		if err != nil {
			if parentJobName != "" {
				return fmt.Errorf("[jobs.%s.step]: %s", parentJobName, err)
			}
			return err
		}

		if jobsToCreate[canonicalName] {
			return nil // Already processed
		}
		jobsToCreate[canonicalName] = true

		// Recursively find all depends_on dependencies
		deps := GetDependencies(job.DependsOn)
		for _, dep := range deps {
			if err := findInvokedJobs(dep, canonicalName); err != nil {
				return err
			}
		}

		// Recursively find all task references
		for _, step := range job.Children() {
			if step.Task != "" {
				if err := findInvokedJobs(step.Task, canonicalName); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Start with jobs in order
	for _, jobName := range jobOrder {
		if err := findInvokedJobs(jobName, ""); err != nil {
			fmt.Printf("%s %s\n", colors.BrightRed("ERROR:"), err)
			os.Exit(1)
		}
	}

	// Create job nodes for all jobs that might be invoked
	// Only add root-level jobs to the tree display; nested jobs are added when invoked as tasks
	jobsToCreateSorted := treeview.SortByOrder(jobsToCreate, jobOrder)
	for _, jobName := range jobsToCreateSorted {
		// Look up job from current pipeline or cross-pipeline jobs
		job := allJobs[jobName]
		if job == nil {
			job = crossPipelineJobs[jobName]
		}
		if job == nil {
			continue // Should not happen if findInvokedJobs worked correctly
		}

		// Display name is the canonical name (already includes skill prefix for cross-pipeline)
		displayName := jobName
		if pipeline.ID != "" && !strings.Contains(jobName, ":") {
			// Only prefix if it's a local job (not already prefixed)
			displayName = pipeline.ID + ":" + jobName
		}

		jobLabel := displayName
		if job.Desc != "" {
			descInterpolated, err := InterpolateString(job.Desc, pipelineCtx)
			if err == nil {
				jobLabel = displayName + " - " + descInterpolated
			} else {
				jobLabel = displayName + " - " + job.Desc
			}
		}

		// Get job dependencies
		deps := GetDependencies(job.DependsOn)

		// Check if this job is in the root execution order
		isRootJob := false
		for _, rootName := range jobOrder {
			if rootName == jobName {
				isRootJob = true
				break
			}
		}

		steps := job.Children()
		isSimpleTask := len(steps) == 1 && len(steps[0].Commands()) > 0 && steps[0].HidePrefix

		// Only add to tree if it's in jobOrder (root-level execution)
		if isRootJob {
			jobNode := tree.AddJobWithoutSteps(deps, jobLabel, job.Nested)
			jobNode.SetSummarize(job.Summarize)

			if !isSimpleTask {
				buildAndAddStepsToJob(jobNode, steps)
			}

			jobNodes[jobName] = jobNode
		} else {
			// For non-root jobs (only invoked as tasks), create nodes but don't add to tree
			jobNode := treeview.NewNode(jobLabel)
			jobNode.SetSummarize(job.Summarize)

			if !isSimpleTask {
				buildAndAddStepsToJob(&treeview.TreeNode{Node: jobNode}, steps)
			}

			jobNodes[jobName] = &treeview.TreeNode{Node: jobNode}
		}
	}
	// Pre-attach task job nodes as children of direct task: step nodes.
	// This ensures the initial tree render accounts for the correct line count
	// before execution begins (e.g., "task: test:mergecov" expands its children).
	// Also pre-attach depends_on dependency nodes (mirroring executor behavior).
	preAttached := make(map[string]bool)
	var preAttachTaskSteps func(jobName string)
	preAttachTaskSteps = func(jobName string) {
		if preAttached[jobName] {
			return
		}
		preAttached[jobName] = true

		jobNode := jobNodes[jobName]
		if jobNode == nil {
			return
		}
		job := allJobs[jobName]
		if job == nil {
			job = crossPipelineJobs[jobName]
		}
		if job == nil {
			return
		}
		stepChildren := jobNode.Node.GetChildren()
		for i, step := range job.Children() {
			if step.Task == "" || !step.For.IsEmpty() || i >= len(stepChildren) {
				continue
			}
			taskJob := allJobs[step.Task]
			if taskJob == nil {
				taskJob = crossPipelineJobs[step.Task]
			}
			if taskJob == nil {
				continue
			}
			// Pre-attach depends_on dependency nodes to the step node
			// (the executor does the same at runtime via synthetic depStep calls)
			for _, depName := range GetDependencies(taskJob.DependsOn) {
				if depNode := jobNodes[depName]; depNode != nil {
					stepChildren[i].AddChild(depNode.Node)
				}
			}
			// Pre-attach the task's own job node
			if taskNode := jobNodes[step.Task]; taskNode != nil {
				stepChildren[i].AddChild(taskNode.Node)
				preAttachTaskSteps(step.Task)
			}
		}
	}
	for _, jobName := range jobsToCreateSorted {
		preAttachTaskSteps(jobName)
	}

	pipelineCtx.JobNodes = jobNodes
	display.Render(root)

	executor := NewExecutor()

	// Helper to execute a job (with dependency checking)
	executeJobWithDeps := func(jobName string, job *model.Job) error {
		// Wait for dependencies if any
		deps := GetDependencies(job.DependsOn)
		for _, dep := range deps {
			for !pipelineCtx.IsJobCompleted(dep) {
				time.Sleep(50 * time.Millisecond)
			}
		}

		jobCtx := pipelineCtx.Copy()
		jobCtx.Job = job
		jobCtx.Depth = 1
		jobCtx.StepSequence = 0 // Reset step counter for each job

		// Get pre-created job node and mark it as running
		jobNode := jobNodes[jobName]
		jobNode.SetStatus(treeview.StatusRunning)
		jobCtx.CurrentJob = jobNode

		// Capture job start time
		var jobStartOffset float64
		if logger != nil {
			jobStartOffset = logger.GetElapsed()
		}
		jobNode.SetStartOffset(jobStartOffset)
		jobStartTime := time.Now()

		display.Render(root)

		execErr := executor.ExecuteJob(ctx, jobCtx)

		// Calculate job duration
		jobDuration := time.Since(jobStartTime)
		jobNode.SetDuration(jobDuration.Seconds())

		// Handle job-level if condition skip
		if errors.Is(execErr, ErrJobSkipped) {
			jobNode.SetStatus(treeview.StatusSkipped)
			if !job.If.IsEmpty() {
				jobNode.SetIf(job.If.String())
			}
			// Mark child steps as skipped too
			for _, child := range jobNode.GetChildren() {
				child.Node.SetStatus(treeview.StatusSkipped)
			}
			display.Render(root)

			// Log skip event
			jobID := "jobs." + jobName
			if logger != nil {
				logger.LogExec(eventlog.ResultSkipped, jobID, jobName, jobStartOffset, jobDuration.Milliseconds(), nil)
			}

			pipelineCtx.MarkJobCompleted(jobName)
			return nil
		}

		// Log job event
		jobID := "jobs." + jobName
		if logger != nil {
			result := eventlog.ResultPass
			if execErr != nil {
				result = eventlog.ResultFail
			}
			logger.LogExec(result, jobID, jobName, jobStartOffset, jobDuration.Milliseconds(), execErr)
		}

		if execErr != nil {
			pipelineCtx.MarkJobCompleted(jobName)
			return execErr
		}

		// Mark job as passed
		jobNode.SetStatus(treeview.StatusPassed)
		display.Render(root)

		pipelineCtx.MarkJobCompleted(jobName)

		return nil
	}

	eg := new(errgroup.Group)
	detached := 0

	for _, name := range jobOrder {
		job := allJobs[name]

		if job == nil {
			return fmt.Errorf("job %q not found in pipeline", name)
		}

		if job.Detach {
			detached++
			// Capture job and name by value to avoid closure variable capture issues
			jobCopy := job
			nameCopy := name
			eg.Go(func() error {
				return executeJobWithDeps(nameCopy, jobCopy)
			})
			continue
		}

		if err := executeJobWithDeps(name, job); err != nil {
			root.SetStatus(treeview.StatusFailed)

			// Clear the live tree and print final scrollable output
			display.RenderFinal(root)

			// Write event log on failure
			writeEventLog(logger, root, err)

			return err
		}
	}

	// Wait for all detached jobs
	var runErr error
	if detached > 0 {
		if err := eg.Wait(); err != nil {
			// Mark pipeline as failed
			root.SetStatus(treeview.StatusFailed)
			runErr = err
		}
	}

	if runErr == nil {
		// Mark pipeline as passed and render final tree
		root.SetStatus(treeview.StatusPassed)
	}

	// Clear the live tree and print final scrollable output
	if !silentOutput {
		display.RenderFinal(root)
	}

	// Write event log
	writeEventLog(logger, root, runErr)

	// Output JSON/YAML if requested
	if silentOutput {
		state := eventlog.NodeToStateNode(root)
		if outputJSON {
			data, _ := json.MarshalIndent(state, "", "  ")
			fmt.Println(string(data))
		} else if outputYAML {
			data, _ := yaml.Marshal(state)
			fmt.Print(string(data))
		}
	}

	return runErr
}

// writeEventLog writes the final event log to the file.
func writeEventLog(logger *eventlog.Logger, root *treeview.Node, runErr error) {
	if logger == nil {
		return
	}

	// Set root duration
	root.SetDuration(logger.GetElapsed())

	// Convert tree to state
	state := eventlog.NodeToStateNode(root)

	// Count steps and build summary
	total, passed, failed, skipped := eventlog.CountSteps(state)

	result := eventlog.ResultPass
	if runErr != nil || failed > 0 {
		result = eventlog.ResultFail
	}

	stats := eventlog.CaptureRuntimeStats()
	summary := &eventlog.RunSummary{
		Duration:     logger.GetElapsed(),
		TotalSteps:   total,
		PassedSteps:  passed,
		FailedSteps:  failed,
		SkippedSteps: skipped,
		Result:       result,
		MemoryAlloc:  stats.MemoryAlloc,
		Goroutines:   stats.Goroutines,
	}

	_ = logger.Write(state, summary)
}

func parseEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return "", ""
}

// printAvailableJobs prints available jobs in a format similar to task.
func printAvailableJobs(jobs map[string]*model.Job, pipelineID string) {
	names := treeview.SortJobsByDepth(jobNames(jobs))
	maxLen := 0
	for _, name := range names {
		displayName := name
		if pipelineID != "" {
			displayName = pipelineID + ":" + name
		}
		if len(displayName) > maxLen {
			maxLen = len(displayName)
		}
	}

	for _, name := range names {
		job := jobs[name]
		if !job.IsRootLevel() {
			continue // Skip nested jobs
		}

		displayName := name
		if pipelineID != "" {
			displayName = pipelineID + ":" + name
		}

		padding := maxLen - len(displayName) + 2
		if job.Desc != "" {
			fmt.Fprintf(os.Stderr, "* %s:%*s%s\n", colors.BrightGreen(displayName), padding, "", job.Desc)
		} else {
			fmt.Fprintf(os.Stderr, "* %s:\n", colors.BrightGreen(displayName))
		}
	}
}
