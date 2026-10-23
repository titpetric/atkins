package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// PipelineOptions contains options for running a pipeline.
type PipelineOptions struct {
	Jobs         []string // Jobs to run (in order)
	LogFile      string
	PipelineFile string
	Debug        bool
	FinalOnly    bool
	Silent       bool
	JSON         bool
	YAML         bool
	AllPipelines []*model.Pipeline // All loaded pipelines for cross-pipeline task references
	Progress     ProgressObserver  // Optional observer for job progress events

	// Transcript receives the finished tree, the same rendering the
	// terminal ends with. A caller uses it to record the run
	// somewhere else - a CI job log - without changing what a person
	// watching it sees.
	Transcript io.Writer

	// Args are the command line arguments after a bare --, handed to
	// the requested jobs. Only the exec: steps of a job with args: true
	// use them.
	Args []string

	// Recorder stores the finished run. Nil records nothing, which is
	// what a test and a linting pass want; the command line hands one
	// over so every run leaves a row behind.
	Recorder Recorder
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

// RunPipeline runs a pipeline with the given options.
func RunPipeline(ctx context.Context, pipeline *model.Pipeline, opts PipelineOptions) error {
	return NewPipeline(pipeline, opts).Run(ctx)
}

// Run returns the error of the first job that fails, nil when every job passes.
func (p *Pipeline) Run(ctx context.Context) error {
	logger := eventlog.NewLogger(p.opts.LogFile, p.data.Name, p.opts.PipelineFile, p.opts.Debug)
	return p.runPipeline(ctx, logger)
}

func (p *Pipeline) runPipeline(ctx context.Context, logger *eventlog.Logger) error {
	var (
		pipeline     = p.data
		jobs         = p.opts.Jobs
		finalOnly    = p.opts.FinalOnly
		outputJSON   = p.opts.JSON
		outputYAML   = p.opts.YAML
		silentOutput = p.opts.Silent || outputJSON || outputYAML
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
		Progress:     p.opts.Progress,
	}

	// Copy environment variables from OS
	for _, env := range os.Environ() {
		k, v := parseEnv(env)
		if k != "" {
			pipelineCtx.Env[k] = v
		}
	}

	// Resolve pipeline-level working directory BEFORE merging variables,
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

	plan, err := p.planJobs(pipelineCtx, tree, silentOutput)
	if err != nil {
		return err
	}
	allJobs, jobOrder, jobNodes := plan.allJobs, plan.jobOrder, plan.jobNodes

	// Build ancestor map: for each dependency job, track which parent chain caused it.
	// e.g. if "default" depends_on "fmt", depAncestors["fmt"] = ["default"].
	depAncestors := buildDepAncestors(allJobs, jobs)

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
		if slices.Contains(jobs, jobName) {
			jobCtx.Args = p.opts.Args
		}
		jobCtx.Depth = 1
		jobCtx.StepSequence = 0 // Reset step counter for each job

		// Set parent chain from dependency ancestry
		ancestors := depAncestors[jobName]
		jobCtx.Parents = append(ancestors, jobName)

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

		pipelineCtx.EmitProgress(JobProgressEvent{
			JobName:   jobName,
			Parents:   ancestors,
			Status:    JobProgressRunning,
			StartedAt: jobStartTime,
		})

		display.Render(root)

		execErr := executor.ExecuteJob(ctx, jobCtx)

		// Calculate job duration
		jobDuration := time.Since(jobStartTime)
		jobNode.SetDuration(jobDuration.Seconds())

		// Handle job-level if condition skip
		if errors.Is(execErr, ErrJobSkipped) {
			jobNode.SetStatus(treeview.StatusSkipped)
			if !job.If.IsEmpty() {
				jobNode.SetIf(interpolateIfDisplay(job.If.String(), jobCtx))
			}
			// Mark child steps as skipped too
			for _, child := range jobNode.GetChildren() {
				child.Node.SetStatus(treeview.StatusSkipped)
			}
			display.Render(root)

			pipelineCtx.EmitProgress(JobProgressEvent{
				JobName:   jobName,
				Parents:   ancestors,
				Status:    JobProgressSkipped,
				StartedAt: jobStartTime,
				Duration:  jobDuration,
			})

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
			pipelineCtx.EmitProgress(JobProgressEvent{
				JobName:   jobName,
				Parents:   ancestors,
				Status:    JobProgressFailed,
				StartedAt: jobStartTime,
				Duration:  jobDuration,
				Err:       execErr,
			})

			pipelineCtx.MarkJobCompleted(jobName)
			return execErr
		}

		// Mark job as passed
		jobNode.SetStatus(treeview.StatusPassed)
		display.Render(root)

		pipelineCtx.EmitProgress(JobProgressEvent{
			JobName:   jobName,
			Parents:   ancestors,
			Status:    JobProgressPassed,
			StartedAt: jobStartTime,
			Duration:  jobDuration,
		})

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
			if !silentOutput {
				display.RenderFinal(root)
			}

			p.writeTranscript(root)

			// Write event log on failure
			p.finishEventLog(ctx, logger, root, err)

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

	p.writeTranscript(root)

	// Write event log
	p.finishEventLog(ctx, logger, root, runErr)

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

// buildDepAncestors walks the depends_on graph for each requested job
// and returns a map from each dependency job name to its ancestor chain.
// For example, if "default" depends_on ["fmt"] and "fmt" depends_on ["lint"],
// the result is: {"fmt": ["default"], "lint": ["default", "fmt"]}.
// Root jobs (the ones directly requested) have no ancestors (empty slice).
func buildDepAncestors(allJobs map[string]*model.Job, requestedJobs []string) map[string][]string {
	ancestors := make(map[string][]string)

	var walk func(jobName string, chain []string)
	walk = func(jobName string, chain []string) {
		job := allJobs[jobName]
		if job == nil {
			return
		}
		for _, dep := range GetDependencies(job.DependsOn) {
			depChain := make([]string, len(chain))
			copy(depChain, chain)
			ancestors[dep] = depChain
			walk(dep, append(depChain, dep))
		}
	}

	for _, name := range requestedJobs {
		walk(name, []string{name})
	}
	return ancestors
}

func parseEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return "", ""
}
