package agent

import (
	"context"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// AutoFixer handles automatic error recovery.
type AutoFixer struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline
}

// NewAutoFixer creates a new auto-fixer.
func NewAutoFixer(resolver *runner.TaskResolver, skills []*model.Pipeline) *AutoFixer {
	return &AutoFixer{
		resolver: resolver,
		skills:   skills,
	}
}

// CanFix checks if a fix skill exists for the given task.
// e.g., if "go:test" fails, checks for "go:fix".
func (f *AutoFixer) CanFix(task *model.ResolvedTask) bool {
	if task == nil || task.Pipeline == nil {
		return false
	}

	skillID := task.Pipeline.ID
	if skillID == "" {
		return false
	}

	fixName := skillID + ":fix"
	_, err := f.resolver.Resolve(fixName)
	return err == nil
}

// GetFixTask returns the fix task for a given skill.
// "go:test" -> "go:fix", "docker:build" -> "docker:fix".
func (f *AutoFixer) GetFixTask(task *model.ResolvedTask) (*model.ResolvedTask, error) {
	if task == nil || task.Pipeline == nil {
		return nil, nil
	}

	skillID := task.Pipeline.ID
	if skillID == "" {
		return nil, nil
	}

	fixName := skillID + ":fix"
	return f.resolver.Resolve(fixName)
}

// AttemptFix runs the fix task and reports success.
func (f *AutoFixer) AttemptFix(ctx context.Context, task *model.ResolvedTask, allPipelines []*model.Pipeline) error {
	fixTask, err := f.GetFixTask(task)
	if err != nil {
		return err
	}
	if fixTask == nil {
		return nil
	}

	// Run the fix task
	err = runner.RunPipeline(ctx, fixTask.Pipeline, runner.PipelineOptions{
		Jobs:         []string{fixTask.Job.Name},
		FinalOnly:    true,
		AllPipelines: allPipelines,
	})

	return err
}

// AutoFixConfig holds configuration for auto-fix behavior.
type AutoFixConfig struct {
	// Enabled controls whether auto-fix is attempted.
	Enabled bool

	// MaxRetries is the maximum number of fix+retry cycles.
	MaxRetries int

	// FixTaskSuffix is the suffix used to find fix tasks (default: "fix").
	FixTaskSuffix string
}

// DefaultAutoFixConfig returns the default auto-fix configuration.
func DefaultAutoFixConfig() *AutoFixConfig {
	return &AutoFixConfig{
		Enabled:       true,
		MaxRetries:    1,
		FixTaskSuffix: "fix",
	}
}
