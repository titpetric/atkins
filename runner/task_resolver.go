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
	Pipeline *model.Pipeline // The pipeline containing the task
	Job      *model.Job      // The resolved job
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

// Resolve resolves a task name to its pipeline and job.
func (r *TaskResolver) Resolve(taskName string) (*ResolvedTask, error) {
	var strict bool
	if strings.HasPrefix(taskName, ":") {
		strict = true
		taskName = taskName[1:]
	}
	return r.resolve(taskName, strict)
}

func (r *TaskResolver) resolve(name string, strict bool) (*ResolvedTask, error) {
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
			return &ResolvedTask{Pipeline: pipeline, Name: name}, true
		}
	}

	return nil, false
}

// resolveAlias checks if alias matches any job alias.
func (r *TaskResolver) resolveAlias(alias string) (*ResolvedTask, bool) {
	for _, pipeline := range r.pipelines {
		aliases := pipeline.GetAliases()
		if target, ok := aliases[alias]; ok {
			return &ResolvedTask{Pipeline: pipeline, Name: target}, true
		}
	}
	return nil, false
}

// resolveFuzzyTarget handles fuzzy/substring matching across all pipelines.
func (r *TaskResolver) resolveFuzzy(name string) (*ResolvedTask, error) {
	matches := findFuzzyMatches(r.pipelines, name)
	if len(matches) == 1 {
		match := matches[0]
		return &ResolvedTask{Pipeline: match.Pipeline, Name: match.Name}, nil
	}
	if len(matches) > 1 {
		return nil, &FuzzyMatchError{Matches: matches}
	}
	return nil, fmt.Errorf("job %q not found", name)
}
