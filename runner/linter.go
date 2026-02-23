package runner

import (
	"fmt"

	"github.com/titpetric/atkins/model"
)

// LintError represents a linting error.
type LintError struct {
	Job    string
	Issue  string
	Detail string
}

// Linter validates a pipeline for correctness.
type Linter struct {
	pipeline     *model.Pipeline
	allPipelines []*model.Pipeline // All pipelines for cross-pipeline validation
	errors       []LintError
}

// NewLinter creates a new linter.
func NewLinter(pipeline *model.Pipeline) *Linter {
	return &Linter{
		pipeline: pipeline,
		errors:   make([]LintError, 0),
	}
}

// NewLinterWithPipelines creates a linter with access to all pipelines for cross-pipeline validation.
func NewLinterWithPipelines(pipeline *model.Pipeline, allPipelines []*model.Pipeline) *Linter {
	return &Linter{
		pipeline:     pipeline,
		allPipelines: allPipelines,
		errors:       make([]LintError, 0),
	}
}

// Lint validates the pipeline and returns any errors.
func (l *Linter) Lint() []LintError {
	l.validateDependencies()
	l.validateTaskInvocations()
	return l.errors
}

// validateDependencies checks that all depends_on references exist
func (l *Linter) validateDependencies() {
	jobs := l.pipeline.Jobs
	if len(jobs) == 0 {
		jobs = l.pipeline.Tasks
	}

	for jobName, job := range jobs {
		if job == nil {
			continue
		}

		deps := GetDependencies(job.DependsOn)
		for _, dep := range deps {
			if _, exists := jobs[dep]; !exists {
				l.errors = append(l.errors, LintError{
					Job:    jobName,
					Issue:  "missing dependency",
					Detail: fmt.Sprintf("job '%s' depends_on '%s', but job '%s' not found", jobName, dep, dep),
				})
			}
		}
	}
}

// validateTaskInvocations checks that referenced tasks exist
func (l *Linter) validateTaskInvocations() {
	jobs := l.pipeline.Jobs
	if len(jobs) == 0 {
		jobs = l.pipeline.Tasks
	}

	for jobName, job := range jobs {
		if job == nil {
			continue
		}

		// Check if both steps and cmds are defined (warning)
		if len(job.Steps) > 0 && len(job.Cmds) > 0 {
			l.errors = append(l.errors, LintError{
				Job:    jobName,
				Issue:  "ambiguous step definition",
				Detail: fmt.Sprintf("job '%s' defines both 'steps' and 'cmds', only 'steps' will be used (cmds is ignored)", jobName),
			})
		}

		// Check each step for task references
		for _, step := range job.Children() {
			if step != nil && step.Task != "" {
				if err := l.validateTaskReference(step.Task); err != nil {
					l.errors = append(l.errors, LintError{
						Job:    jobName,
						Issue:  "missing task reference",
						Detail: err.Error(),
					})
				}
			}
		}
	}
}

// validateTaskReference validates a task reference using the shared TaskResolver.
func (l *Linter) validateTaskReference(taskName string) error {
	resolver := &TaskResolver{
		CurrentPipeline: l.pipeline,
		AllPipelines:    l.allPipelines,
	}
	if err := resolver.Validate(taskName); err != nil {
		return fmt.Errorf("step references task '%s', but %s", taskName, err)
	}
	return nil
}

// GetDependencies converts depends_on field (string or []string) to a slice of job names.
func GetDependencies(dependsOn any) []string {
	if dependsOn == nil {
		return []string{}
	}

	switch v := dependsOn.(type) {
	case string:
		return []string{v}
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return v
	case model.Dependencies:
		return []string(v)
	default:
		return []string{}
	}
}

// NoDefaultJobError is returned when no default job is found.
type NoDefaultJobError struct {
	Jobs map[string]*model.Job
}

// Error returns the error hinting a default job should be defined.
func (e *NoDefaultJobError) Error() string {
	return "task \"default\" does not exist"
}

// findDefaultJob finds the "default" job, either directly or via alias.
// Returns the job name to use (which may differ from "default" if found via alias).
func findDefaultJob(jobs map[string]*model.Job) (string, bool) {
	// First check for direct "default" job
	if _, exists := jobs["default"]; exists {
		return "default", true
	}

	// Check for a job with "default" as an alias
	for jobName, job := range jobs {
		for _, alias := range job.Aliases {
			if alias == "default" {
				return jobName, true
			}
		}
	}

	return "", false
}

// ResolveJobDependencies returns jobs in dependency order.
// Returns the jobs to run and any resolution errors.
func ResolveJobDependencies(jobs map[string]*model.Job, startingJob string) ([]string, error) {
	if len(jobs) == 0 {
		return []string{}, nil
	}

	// If a specific job is requested, resolve its dependency chain
	if startingJob != "" {
		if _, exists := jobs[startingJob]; !exists {
			return nil, fmt.Errorf("job '%s' not found", startingJob)
		}
		return resolveDependencyChain(jobs, startingJob)
	}

	// Otherwise, resolve root jobs (those without ':' in name)
	// Mark nested jobs so they won't be executed directly
	for name, job := range jobs {
		if job.Name == "" {
			job.Name = name
		}
		if !job.IsRootLevel() {
			job.Nested = true
		}
	}

	// If 'default' job exists (directly or via alias), start with that
	if defaultJob, found := findDefaultJob(jobs); found {
		return resolveDependencyChain(jobs, defaultJob)
	}

	// No default job found - return error with available jobs
	return nil, &NoDefaultJobError{Jobs: jobs}
}

// resolveDependencyChain returns a job and all its dependencies in execution order
func resolveDependencyChain(jobs map[string]*model.Job, jobName string) ([]string, error) {
	// Set Name field on all jobs for IsRootLevel() check
	for name, job := range jobs {
		if job.Name == "" {
			job.Name = name
		}
	}

	resolved := make([]string, 0)
	visited := make(map[string]bool)
	var visit func(string) error

	visit = func(name string) error {
		if visited[name] {
			return nil // Already visited
		}

		job, exists := jobs[name]
		if !exists {
			return fmt.Errorf("job '%s' not found", name)
		}

		visited[name] = true

		// Visit dependencies first
		deps := GetDependencies(job.DependsOn)
		for _, dep := range deps {
			if err := visit(dep); err != nil {
				return err
			}
		}

		resolved = append(resolved, name)
		return nil
	}

	if err := visit(jobName); err != nil {
		return nil, err
	}

	return resolved, nil
}

// ValidateJobRequirements checks that all required variables are present in the context.
// Returns an error with a clear message listing missing variables.
func ValidateJobRequirements(ctx *ExecutionContext, job *model.Job) error {
	if len(job.Requires) == 0 {
		return nil // No requirements to validate
	}

	var missing []string
	for _, varName := range job.Requires {
		if _, exists := ctx.Variables[varName]; !exists {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("job '%s' requires variables %v but missing: %v", job.Name, job.Requires, missing)
	}

	return nil
}
