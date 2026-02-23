package runner

import (
	"fmt"
	"strings"

	"github.com/titpetric/atkins/model"
)

// ResolvedTask contains the result of resolving a task reference.
type ResolvedTask struct {
	Name     string          // Canonical name (e.g., "go:build" or "build")
	Pipeline *model.Pipeline // The pipeline containing the task
	Job      *model.Job      // The resolved job
}

// TaskResolver resolves task references, handling cross-pipeline : prefix syntax.
// Supports:
//   - ":build" → main pipeline (ID="") job "build".
//   - ":go:build" → skill "go" job "build".
//   - "build" → current pipeline job "build".
type TaskResolver struct {
	CurrentPipeline *model.Pipeline
	AllPipelines    []*model.Pipeline
}

// Resolve resolves a task name to its pipeline and job.
// Returns an error if the task cannot be found.
func (r *TaskResolver) Resolve(taskName string) (*ResolvedTask, error) {
	// Check for explicit cross-pipeline reference (leading colon)
	if strings.HasPrefix(taskName, ":") {
		// If no pipelines are loaded, fall back to local lookup (graceful degradation)
		if len(r.AllPipelines) == 0 {
			return r.resolveLocal(taskName[1:])
		}
		return r.resolveCrossPipeline(taskName[1:]) // Remove leading colon
	}

	// Regular task reference - look in current pipeline
	return r.resolveLocal(taskName)
}

// resolveCrossPipeline resolves a task with explicit pipeline reference.
// Handles ":skillID:jobName" and ":jobName" (main pipeline) formats.
func (r *TaskResolver) resolveCrossPipeline(explicitName string) (*ResolvedTask, error) {
	// Check if it's :skillID:jobName or just :jobName
	if parts := strings.SplitN(explicitName, ":", 2); len(parts) == 2 {
		// :go:build → skill "go", job "build"
		skillID, jobName := parts[0], parts[1]
		return r.resolveInSkill(skillID, jobName)
	}

	// :build → main pipeline (ID="") job "build"
	return r.resolveInMain(explicitName)
}

// resolveInSkill looks up a job in a specific skill pipeline.
func (r *TaskResolver) resolveInSkill(skillID, jobName string) (*ResolvedTask, error) {
	for _, p := range r.AllPipelines {
		if p.ID == skillID {
			jobs := getJobsFromPipeline(p)
			if job, exists := jobs[jobName]; exists {
				return &ResolvedTask{
					Name:     skillID + ":" + jobName,
					Pipeline: p,
					Job:      job,
				}, nil
			}
			return nil, fmt.Errorf("task %q not found in skill %q", jobName, skillID)
		}
	}
	return nil, fmt.Errorf("skill %q not found", skillID)
}

// resolveInMain looks up a job in the main pipeline (ID="").
func (r *TaskResolver) resolveInMain(jobName string) (*ResolvedTask, error) {
	for _, p := range r.AllPipelines {
		if p.ID == "" {
			jobs := getJobsFromPipeline(p)
			if job, exists := jobs[jobName]; exists {
				return &ResolvedTask{
					Name:     jobName,
					Pipeline: p,
					Job:      job,
				}, nil
			}
			return nil, fmt.Errorf("task %q not found in main pipeline", jobName)
		}
	}
	return nil, fmt.Errorf("main pipeline not found")
}

// resolveLocal looks up a job in the current pipeline.
func (r *TaskResolver) resolveLocal(taskName string) (*ResolvedTask, error) {
	if r.CurrentPipeline == nil {
		return nil, fmt.Errorf("no current pipeline set")
	}

	jobs := getJobsFromPipeline(r.CurrentPipeline)
	if job, exists := jobs[taskName]; exists {
		return &ResolvedTask{
			Name:     taskName,
			Pipeline: r.CurrentPipeline,
			Job:      job,
		}, nil
	}
	return nil, fmt.Errorf("task %q not found in pipeline", taskName)
}

// Validate checks if a task reference is valid without returning the full result.
// This is useful for linting where you only need to know if the reference is valid.
func (r *TaskResolver) Validate(taskName string) error {
	_, err := r.Resolve(taskName)
	return err
}

// getJobsFromPipeline returns jobs from a pipeline, falling back to tasks if empty.
// This consolidates the common pattern used throughout the codebase.
func getJobsFromPipeline(p *model.Pipeline) map[string]*model.Job {
	if len(p.Jobs) > 0 {
		return p.Jobs
	}
	return p.Tasks
}
