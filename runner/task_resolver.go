package runner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/titpetric/atkins/model"
)

// ResolvedTask contains the result of resolving a task reference.
type ResolvedTask struct {
	Name     string          // Canonical name (e.g., "go:build" or "build")
	Job      *model.Job      // The resolved job
	Pipeline *model.Pipeline // The pipeline containing the task
}

// NewResolvedTask creates a ResolvedTask with all required fields.
func NewResolvedTask(pipeline *model.Pipeline, job *model.Job, name string) *ResolvedTask {
	return &ResolvedTask{
		Name:     name,
		Job:      job,
		Pipeline: pipeline,
	}
}

// TaskResolver resolves task references, handling cross-pipeline : prefix syntax.
type TaskResolver struct {
	pipelines []*model.Pipeline
}

// NewTaskResolver will provide a task resolver for a set of pipelines.
func NewTaskResolver(pipelines []*model.Pipeline) *TaskResolver {
	return &TaskResolver{
		pipelines: pipelines,
	}
}

// NewSkillResolver will provide a task resolver for a skill pipeline.
func NewSkillResolver(pipeline *model.Pipeline) *TaskResolver {
	return &TaskResolver{
		pipelines: []*model.Pipeline{pipeline},
	}
}

// Resolve resolves a task name to its pipeline and job.
func (r *TaskResolver) Resolve(taskName string) (*ResolvedTask, error) {
	var strict bool
	if strings.HasPrefix(taskName, ":") {
		strict = true
		taskName = taskName[1:]
	}
	return r.ResolveName(taskName, strict)
}

// ResolveName resolves a name to the pipeline and job.
// It tries explicit matching, then checks aliases, then fuzzy matches jobs.
// If no job is matched, an error is returned.
func (r *TaskResolver) ResolveName(name string, strict bool) (*ResolvedTask, error) {
	if target, found := r.resolveExplicitTarget(name); found {
		return target, nil
	}
	if !strict {
		if target, found := r.resolveAlias(name); found {
			return target, nil
		}

		return r.resolveFuzzy(name)
	}

	return nil, fmt.Errorf("task %q not resolved (strict: %v)", name, strict)
}

// resolveExplicitTarget should iterate each pipelines available targets for
// an exact match. If no match is found, a nil, false is returned.
func (r *TaskResolver) resolveExplicitTarget(name string) (*ResolvedTask, bool) {
	// exact name match across all pipelines.
	for _, pipeline := range r.pipelines {
		keys := pipeline.GetKeys()
		if slices.Contains(keys, name) {
			return NewResolvedTask(pipeline, lookupJob(pipeline, name), name), true
		}
	}

	return nil, false
}

// lookupJob finds a job in the pipeline by its canonical name, stripping
// the pipeline ID prefix if present.
func lookupJob(pipeline *model.Pipeline, name string) *model.Job {
	jobs := pipeline.GetJobs()
	// Strip pipeline ID prefix if present (e.g., "go:build" -> "build")
	if pipeline.ID != "" {
		name = strings.TrimPrefix(name, pipeline.ID+":")
	}
	return jobs[name]
}

// resolveAlias checks if alias matches any job alias.
func (r *TaskResolver) resolveAlias(alias string) (*ResolvedTask, bool) {
	for _, pipeline := range r.pipelines {
		aliases := pipeline.GetAliases()
		if target, ok := aliases[alias]; ok {
			return NewResolvedTask(pipeline, lookupJob(pipeline, target), target), true
		}
	}
	return nil, false
}

// resolveFuzzyTarget handles fuzzy/substring matching across all pipelines.
func (r *TaskResolver) resolveFuzzy(name string) (*ResolvedTask, error) {
	matches := findFuzzyMatches(r.pipelines, name)
	if len(matches) == 1 {
		match := matches[0]
		return NewResolvedTask(match.Pipeline, match.Job, match.Name), nil
	}
	if len(matches) > 1 {
		return nil, &FuzzyMatchError{Matches: matches}
	}
	return nil, fmt.Errorf("job %q not found", name)
}
