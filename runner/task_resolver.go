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
//   - "go:build" → skill "go" job "build" (fallback when not found locally).
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
			jobs := getJobs(p)
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
			jobs := getJobs(p)
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
// If not found and taskName contains a colon (skill:job format), falls back to skill lookup.
func (r *TaskResolver) resolveLocal(taskName string) (*ResolvedTask, error) {
	if r.CurrentPipeline == nil {
		return nil, fmt.Errorf("no current pipeline set")
	}

	jobs := getJobs(r.CurrentPipeline)
	if job, exists := jobs[taskName]; exists {
		return &ResolvedTask{
			Name:     taskName,
			Pipeline: r.CurrentPipeline,
			Job:      job,
		}, nil
	}

	// If taskName contains a colon and not found locally, try skill resolution
	// This supports "compose:up" as a shorthand for ":compose:up"
	if strings.Contains(taskName, ":") && len(r.AllPipelines) > 0 {
		if resolved, err := r.resolveCrossPipeline(taskName); err == nil {
			return resolved, nil
		}
	}

	return nil, fmt.Errorf("task %q not found in pipeline", taskName)
}

// Validate checks if a task reference is valid without returning the full result.
// This is useful for linting where you only need to know if the reference is valid.
func (r *TaskResolver) Validate(taskName string) error {
	_, err := r.Resolve(taskName)
	return err
}

// ResolvedJobTarget is the result of resolving a job target from the command line.
type ResolvedJobTarget struct {
	Pipeline *model.Pipeline
	JobName  string
}

// ResolveJobTarget determines which pipeline and job to run based on the job name.
// Resolution order:
// 1. Invoked pipeline (`:` prefix) - directly invoke job, bypassing aliases
// 2. Main pipeline exact match - job name exactly matches in main pipeline
// 3. Main pipeline alias - alias matches in main pipeline
// 4. Skills exact match - job name matches in a skill pipeline
// 5. Prefixed job (e.g., "go:test") - explicit skill:job reference
// 6. Skills alias - alias matches in a skill pipeline
// 7. Fuzzy match - suffix/substring match in job names (if exactly one match)
// If no match found, returns error.
func (r *TaskResolver) ResolveJobTarget(jobName string) (*ResolvedJobTarget, error) {
	// 1. Invoked pipeline (`:` prefix)
	if target, found, err := r.resolveExplicitTarget(jobName); found || err != nil {
		return target, err
	}

	// 2. Exact match in main pipeline
	if target, found := r.resolveMainExact(jobName); found {
		return target, nil
	}

	// 3. Main pipeline alias
	if target, found := r.resolveMainAlias(jobName); found {
		return target, nil
	}

	// 4. Skills exact match
	if target, found := r.resolveSkillExact(jobName); found {
		return target, nil
	}

	// 5. Prefixed job (e.g., "go:test")
	if target, found := r.resolvePrefixedSkillTarget(jobName); found {
		return target, nil
	}

	// 6. Skills alias
	if target, found := r.resolveSkillAlias(jobName); found {
		return target, nil
	}

	// 7. Fuzzy match
	return r.resolveFuzzyTarget(jobName)
}

// mainPipeline finds the pipeline with ID="".
func (r *TaskResolver) mainPipeline() *model.Pipeline {
	for _, p := range r.AllPipelines {
		if p.ID == "" {
			return p
		}
	}
	return nil
}

// pipelineByID finds a pipeline by its ID.
func (r *TaskResolver) pipelineByID(id string) *model.Pipeline {
	for _, p := range r.AllPipelines {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// resolveExactInPipeline checks if jobName exists as an exact job in the pipeline.
func (r *TaskResolver) resolveExactInPipeline(p *model.Pipeline, jobName string) (*ResolvedJobTarget, bool) {
	jobs := getJobs(p)
	if _, exists := jobs[jobName]; exists {
		return &ResolvedJobTarget{Pipeline: p, JobName: jobName}, true
	}
	return nil, false
}

// resolveAliasInPipeline checks if alias matches any job alias in the pipeline.
func (r *TaskResolver) resolveAliasInPipeline(p *model.Pipeline, alias string) (*ResolvedJobTarget, bool) {
	jobs := getJobs(p)
	for jn, job := range jobs {
		for _, a := range job.Aliases {
			if a == alias {
				return &ResolvedJobTarget{Pipeline: p, JobName: jn}, true
			}
		}
	}
	return nil, false
}

// resolveExplicitTarget handles `:` prefix - directly invoke job, bypassing aliases.
func (r *TaskResolver) resolveExplicitTarget(jobName string) (*ResolvedJobTarget, bool, error) {
	if !strings.HasPrefix(jobName, ":") {
		return nil, false, nil
	}
	explicitName := jobName[1:]

	if parts := strings.SplitN(explicitName, ":", 2); len(parts) == 2 {
		skillID, skillJob := parts[0], parts[1]
		if p := r.pipelineByID(skillID); p != nil {
			return &ResolvedJobTarget{Pipeline: p, JobName: skillJob}, true, nil
		}
		return nil, true, fmt.Errorf("skill %q not found", skillID)
	}

	if p := r.mainPipeline(); p != nil {
		return &ResolvedJobTarget{Pipeline: p, JobName: explicitName}, true, nil
	}
	return nil, true, fmt.Errorf("main pipeline not found")
}

// resolveMainExact checks for exact match in the main pipeline.
func (r *TaskResolver) resolveMainExact(jobName string) (*ResolvedJobTarget, bool) {
	if p := r.mainPipeline(); p != nil {
		return r.resolveExactInPipeline(p, jobName)
	}
	return nil, false
}

// resolveMainAlias checks for alias match in the main pipeline.
func (r *TaskResolver) resolveMainAlias(jobName string) (*ResolvedJobTarget, bool) {
	if p := r.mainPipeline(); p != nil {
		return r.resolveAliasInPipeline(p, jobName)
	}
	return nil, false
}

// resolveSkillExact checks for exact match in skill pipelines.
func (r *TaskResolver) resolveSkillExact(jobName string) (*ResolvedJobTarget, bool) {
	for _, p := range r.AllPipelines {
		if p.ID != "" {
			if target, found := r.resolveExactInPipeline(p, jobName); found {
				return target, true
			}
		}
	}
	return nil, false
}

// resolvePrefixedSkillTarget handles "go:test" style references.
func (r *TaskResolver) resolvePrefixedSkillTarget(jobName string) (*ResolvedJobTarget, bool) {
	if parts := strings.SplitN(jobName, ":", 2); len(parts) == 2 {
		skillID, skillJob := parts[0], parts[1]
		if p := r.pipelineByID(skillID); p != nil {
			return &ResolvedJobTarget{Pipeline: p, JobName: skillJob}, true
		}
	}
	return nil, false
}

// resolveSkillAlias checks for alias match in skill pipelines.
func (r *TaskResolver) resolveSkillAlias(jobName string) (*ResolvedJobTarget, bool) {
	for _, p := range r.AllPipelines {
		if p.ID != "" {
			if target, found := r.resolveAliasInPipeline(p, jobName); found {
				return target, true
			}
		}
	}
	return nil, false
}

// resolveFuzzyTarget handles fuzzy/substring matching across all pipelines.
func (r *TaskResolver) resolveFuzzyTarget(jobName string) (*ResolvedJobTarget, error) {
	matches := findFuzzyMatches(r.AllPipelines, jobName)
	if len(matches) == 1 {
		match := matches[0]
		return &ResolvedJobTarget{Pipeline: match.Pipeline, JobName: match.JobName}, nil
	}
	if len(matches) > 1 {
		return nil, &FuzzyMatchError{Matches: matches}
	}
	return nil, fmt.Errorf("job %q not found", jobName)
}

// getJobs returns jobs from a pipeline, falling back to tasks if empty.
// This consolidates the common pattern used throughout the codebase.
func getJobs(p *model.Pipeline) map[string]*model.Job {
	if len(p.Jobs) > 0 {
		return p.Jobs
	}
	return p.Tasks
}
