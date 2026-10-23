package runner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/titpetric/atkins/model"
)

// TaskResolver resolves a task reference to a pipeline and a job.
//
// A name resolves by exact name first, a qualified one such as go:build
// included, then by a job alias, then by a unique suffix match. A leading
// colon makes the reference strict: exact names only, and in a skill, the
// global pipelines only. A suffix that matches more than one job is a
// FuzzyMatchError.
type TaskResolver struct {
	pipelines []*model.Pipeline
}

// NewTaskResolver returns a task resolver for a set of pipelines.
func NewTaskResolver(pipelines []*model.Pipeline) *TaskResolver {
	return &TaskResolver{
		pipelines: pipelines,
	}
}

// NewSkillResolver returns a task resolver for a skill pipeline.
func NewSkillResolver(pipeline *model.Pipeline) *TaskResolver {
	return &TaskResolver{
		pipelines: []*model.Pipeline{pipeline},
	}
}

// Resolve resolves a task name to its pipeline and job.
func (r *TaskResolver) Resolve(taskName string) (*model.ResolvedTask, error) {
	var strict bool
	if strings.HasPrefix(taskName, ":") {
		strict = true
		taskName = taskName[1:]
	}
	return r.ResolveName(taskName, strict)
}

// ResolveWithFallback resolves a ":" name with fallback alone,
// any other name with r first and then with fallback.
func (r *TaskResolver) ResolveWithFallback(taskName string, fallback *TaskResolver) (*model.ResolvedTask, error) {
	if strings.HasPrefix(taskName, ":") {
		return fallback.Resolve(taskName)
	}
	resolved, err := r.Resolve(taskName)
	if err != nil {
		resolved, err = fallback.Resolve(taskName)
	}
	return resolved, err
}

// ResolveName returns an exact match, else an alias, else a fuzzy match; strict stops after the exact one.
func (r *TaskResolver) ResolveName(name string, strict bool) (*model.ResolvedTask, error) {
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

// resolveExplicitTarget iterates the available targets of each pipeline for
// an exact match. Without a match it returns nil, false.
func (r *TaskResolver) resolveExplicitTarget(name string) (*model.ResolvedTask, bool) {
	// exact name match across all pipelines.
	for _, pipeline := range r.pipelines {
		keys := pipeline.GetKeys()
		if slices.Contains(keys, name) {
			return model.NewResolvedTask(pipeline, lookupJob(pipeline, name), name), true
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
func (r *TaskResolver) resolveAlias(alias string) (*model.ResolvedTask, bool) {
	for _, pipeline := range r.pipelines {
		aliases := pipeline.GetAliases()
		if target, ok := aliases[alias]; ok {
			return model.NewResolvedTask(pipeline, lookupJob(pipeline, target), target), true
		}
	}
	return nil, false
}

// resolveFuzzyTarget handles fuzzy/substring matching across all pipelines.
func (r *TaskResolver) resolveFuzzy(name string) (*model.ResolvedTask, error) {
	matches := findFuzzyMatches(r.pipelines, name)
	if len(matches) == 1 {
		match := matches[0]
		return model.NewResolvedTask(match.Pipeline, match.Job, match.Name), nil
	}
	if len(matches) > 1 {
		return nil, &FuzzyMatchError{Matches: matches}
	}
	return nil, fmt.Errorf("job %q not found", name)
}
