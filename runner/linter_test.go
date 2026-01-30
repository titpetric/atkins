package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/model"
)

// TestLinter_StepsAndCmdsEquivalence verifies that steps and cmds are treated equivalently
func TestLinter_StepsAndCmdsEquivalence(t *testing.T) {
	// Test with steps
	pipelineWithSteps := &model.Pipeline{
		Name: "test-pipeline",
		Jobs: map[string]*model.Job{
			"test-job": {
				Name: "test-job",
				Steps: []*model.Step{
					{Run: "echo hello"},
				},
			},
		},
	}

	// Test with cmds
	pipelineWithCmds := &model.Pipeline{
		Name: "test-pipeline",
		Jobs: map[string]*model.Job{
			"test-job": {
				Name: "test-job",
				Cmds: []*model.Step{
					{Run: "echo hello"},
				},
			},
		},
	}

	linter1 := NewLinter(pipelineWithSteps)
	errors1 := linter1.Lint()
	assert.Len(t, errors1, 0, "pipeline with steps should have no linter errors")

	linter2 := NewLinter(pipelineWithCmds)
	errors2 := linter2.Lint()
	assert.Len(t, errors2, 0, "pipeline with cmds should have no linter errors")
}

// TestLinter_BothStepsAndCmdsWarning verifies that using both steps and cmds generates a warning
func TestLinter_BothStepsAndCmdsWarning(t *testing.T) {
	pipeline := &model.Pipeline{
		Name: "test-pipeline",
		Jobs: map[string]*model.Job{
			"test-job": {
				Name: "test-job",
				Steps: []*model.Step{
					{Run: "echo from steps"},
				},
				Cmds: []*model.Step{
					{Run: "echo from cmds"},
				},
			},
		},
	}

	linter := NewLinter(pipeline)
	errors := linter.Lint()

	assert.Len(t, errors, 1, "pipeline with both steps and cmds should have exactly one warning")
	assert.Equal(t, errors[0].Job, "test-job")
	assert.Equal(t, errors[0].Issue, "ambiguous step definition")
	assert.Contains(t, errors[0].Detail, "steps")
	assert.Contains(t, errors[0].Detail, "cmds")
}

// TestLinter_MissingTaskReference verifies that linter detects missing task invocations
func TestLinter_MissingTaskReference(t *testing.T) {
	pipeline := &model.Pipeline{
		Name: "test-pipeline",
		Jobs: map[string]*model.Job{
			"test-job": {
				Name: "test-job",
				Steps: []*model.Step{
					{Task: "nonexistent-task"},
				},
			},
			"other-job": {
				Name: "other-job",
				Steps: []*model.Step{
					{Run: "echo hello"},
				},
			},
		},
	}

	linter := NewLinter(pipeline)
	errors := linter.Lint()

	assert.Len(t, errors, 1)
	assert.Equal(t, errors[0].Job, "test-job")
	assert.Equal(t, errors[0].Issue, "missing task reference")
	assert.Contains(t, errors[0].Detail, "nonexistent-task")
}

// TestLinter_MissingDependency verifies that linter detects missing dependencies
func TestLinter_MissingDependency(t *testing.T) {
	pipeline := &model.Pipeline{
		Name: "test-pipeline",
		Jobs: map[string]*model.Job{
			"test-job": {
				Name:      "test-job",
				DependsOn: model.Dependencies{"nonexistent-dep"},
				Steps: []*model.Step{
					{Run: "echo hello"},
				},
			},
		},
	}

	linter := NewLinter(pipeline)
	errors := linter.Lint()

	assert.Len(t, errors, 1)
	assert.Equal(t, errors[0].Job, "test-job")
	assert.Equal(t, errors[0].Issue, "missing dependency")
	assert.Contains(t, errors[0].Detail, "nonexistent-dep")
}

// TestJobChildrenConsistency verifies that Job.Children() is used consistently
func TestJobChildrenConsistency(t *testing.T) {
	// Test that Children() returns Steps when available
	job1 := &model.Job{
		Name: "job1",
		Steps: []*model.Step{
			{Run: "echo from steps"},
		},
	}
	assert.Equal(t, job1.Children(), job1.Steps)

	// Test that Children() returns Cmds when Steps is nil
	job2 := &model.Job{
		Name: "job2",
		Cmds: []*model.Step{
			{Run: "echo from cmds"},
		},
	}
	assert.Equal(t, job2.Children(), job2.Cmds)

	// Test that Children() returns nil when both are nil
	job3 := &model.Job{
		Name: "job3",
	}
	assert.Nil(t, job3.Children())

	// Test that Steps takes precedence over Cmds
	job4 := &model.Job{
		Name: "job4",
		Steps: []*model.Step{
			{Run: "echo from steps"},
		},
		Cmds: []*model.Step{
			{Run: "echo from cmds"},
		},
	}
	assert.Equal(t, job4.Children(), job4.Steps)
	assert.NotEqual(t, job4.Children(), job4.Cmds)
}
