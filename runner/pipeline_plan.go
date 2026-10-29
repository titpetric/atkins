package runner

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/model"
	runnererrors "github.com/titpetric/atkins/runner/errors"
	"github.com/titpetric/atkins/treeview"
)

// jobPlan is what a run resolves before it starts any job: the requested
// jobs and their dependencies in run order, and a tree node for every job a
// step can invoke. The first render shows the whole run.
type jobPlan struct {
	allJobs  map[string]*model.Job
	jobOrder []string
	jobNodes map[string]*treeview.TreeNode
}

// planJobs returns the plan, or the first job that does not resolve.
func (p *Pipeline) planJobs(pipelineCtx *ExecutionContext, tree *treeview.Builder, silentOutput bool) (*jobPlan, error) {
	var (
		pipeline = p.data
		jobs     = p.opts.Jobs
	)

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
				if !silentOutput {
					fmt.Fprintf(os.Stderr, "%s Available jobs for this project:\n", colors.BrightYellow("atkins:"))
					printAvailableJobs(noDefaultErr.Jobs, pipeline.ID)
					fmt.Fprintf(os.Stderr, "%s Job %q does not exist\n", colors.BrightRed("atkins:"), "default")
				}
				return nil, err
			}
			if !silentOutput {
				fmt.Printf("%s %s\n", colors.BrightRed("ERROR:"), err)
			}
			return nil, err
		}
		// Add jobs to unified order, skipping duplicates
		for _, name := range order {
			if !seenJobs[name] {
				seenJobs[name] = true
				jobOrder = append(jobOrder, name)
			}
		}
	}

	// Pre-populate all jobs as pending - include all jobs the run might invoke
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

	// Recursively find all jobs the run might invoke
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

		if job.Inactive {
			err := fmt.Errorf("job %q does not apply here: its when: block does not match", canonicalName)
			if parentJobName != "" {
				return fmt.Errorf("[jobs.%s.step]: %w", parentJobName, err)
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
			for _, taskRef := range step.Tasks() {
				if err := findInvokedJobs(taskRef, canonicalName); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Start with jobs in order
	for _, jobName := range jobOrder {
		if err := findInvokedJobs(jobName, ""); err != nil {
			if !silentOutput {
				fmt.Printf("%s %s\n", colors.BrightRed("ERROR:"), err)
			}
			return nil, err
		}
	}

	// Create job nodes for all jobs the run might invoke
	// Only add root-level jobs to the tree display; a nested job joins when a task invokes it
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

			// The label a person reads carries the description, which an
			// author edits freely. A recorder grouping one job's runs over
			// months needs a key that survives that. The node keeps the
			// job's canonical name in the same scheme its steps use.
			jobNode.SetID("jobs." + displayName)

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
			// A step may reference more than one job (`task: lint test build`);
			// each one goes under the same step node, in order.
			for _, taskRef := range step.Tasks() {
				taskJob := allJobs[taskRef]
				if taskJob == nil {
					taskJob = crossPipelineJobs[taskRef]
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
				if taskNode := jobNodes[taskRef]; taskNode != nil {
					stepChildren[i].AddChild(taskNode.Node)
					preAttachTaskSteps(taskRef)
				}
			}
		}
	}
	for _, jobName := range jobsToCreateSorted {
		preAttachTaskSteps(jobName)
	}

	return &jobPlan{
		allJobs:  allJobs,
		jobOrder: jobOrder,
		jobNodes: jobNodes,
	}, nil
}

// buildAndAddStepsToJob adds step nodes to a job node with command children for multi-command steps
func buildAndAddStepsToJob(jobNode *treeview.TreeNode, steps []*model.Step) {
	for _, step := range steps {
		stepNode := treeview.NewPendingStepNode(step.DisplayLabel(), step.IsDeferred(), step.Summarize)
		stepNode.SetQuiet(step.Quiet)
		// Only add command child nodes if step has more than one command (the label already shows a single command)
		commands := step.Commands()
		if len(commands) > 1 {
			for _, cmd := range commands {
				stepNode.AddChild(treeview.NewCmdNode(cmd))
			}
		}
		jobNode.AddChild(stepNode)
	}
}

// printAvailableJobs prints available jobs in a format like task's.
func printAvailableJobs(jobs map[string]*model.Job, pipelineID string) {
	names := treeview.SortJobsByDepth(slices.Collect(maps.Keys(jobs)))
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
