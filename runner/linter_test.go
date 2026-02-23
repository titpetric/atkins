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

// TestFindDefaultJob verifies default job lookup including alias support
func TestFindDefaultJob(t *testing.T) {
	t.Run("direct default job", func(t *testing.T) {
		jobs := map[string]*model.Job{
			"default": {Name: "default"},
			"build":   {Name: "build"},
		}
		name, found := findDefaultJob(jobs)
		assert.True(t, found)
		assert.Equal(t, "default", name)
	})

	t.Run("default via alias", func(t *testing.T) {
		jobs := map[string]*model.Job{
			"build": {Name: "build", Aliases: []string{"default"}},
			"test":  {Name: "test"},
		}
		name, found := findDefaultJob(jobs)
		assert.True(t, found)
		assert.Equal(t, "build", name)
	})

	t.Run("no default", func(t *testing.T) {
		jobs := map[string]*model.Job{
			"build": {Name: "build"},
			"test":  {Name: "test"},
		}
		_, found := findDefaultJob(jobs)
		assert.False(t, found)
	})

	t.Run("direct default takes precedence over alias", func(t *testing.T) {
		jobs := map[string]*model.Job{
			"default": {Name: "default"},
			"build":   {Name: "build", Aliases: []string{"default"}},
		}
		name, found := findDefaultJob(jobs)
		assert.True(t, found)
		assert.Equal(t, "default", name)
	})
}

// TestNoDefaultJobError verifies error message
func TestNoDefaultJobError(t *testing.T) {
	err := &NoDefaultJobError{
		Jobs: map[string]*model.Job{
			"build": {},
			"test":  {},
		},
	}
	assert.Equal(t, `task "default" does not exist`, err.Error())
}

// TestResolveJobDependencies_NoDefault verifies NoDefaultJobError is returned
func TestResolveJobDependencies_NoDefault(t *testing.T) {
	jobs := map[string]*model.Job{
		"build": {Name: "build"},
		"test":  {Name: "test"},
	}
	_, err := ResolveJobDependencies(jobs, "")
	assert.Error(t, err)
	var noDefaultErr *NoDefaultJobError
	assert.ErrorAs(t, err, &noDefaultErr)
}

// TestLinterWithPipelines_CrossPipelineValidation tests cross-pipeline task validation
func TestLinterWithPipelines_CrossPipelineValidation(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID:   "",
		Name: "main",
		Jobs: map[string]*model.Job{
			"build": {Name: "build", Steps: []*model.Step{{Run: "echo build"}}},
		},
	}

	goSkill := &model.Pipeline{
		ID:   "go",
		Name: "Go Skill",
		Jobs: map[string]*model.Job{
			"test": {
				Name: "test",
				Steps: []*model.Step{
					{Task: ":build"}, // Reference main pipeline
				},
			},
		},
	}

	allPipelines := []*model.Pipeline{mainPipeline, goSkill}

	// Validate goSkill with access to all pipelines
	linter := NewLinterWithPipelines(goSkill, allPipelines)
	errors := linter.Lint()
	assert.Len(t, errors, 0, "valid cross-pipeline reference should not produce errors")
}

// TestLinterWithPipelines_InvalidCrossPipelineRef tests invalid cross-pipeline references
func TestLinterWithPipelines_InvalidCrossPipelineRef(t *testing.T) {
	mainPipeline := &model.Pipeline{
		ID:   "",
		Name: "main",
		Jobs: map[string]*model.Job{},
	}

	goSkill := &model.Pipeline{
		ID:   "go",
		Name: "Go Skill",
		Jobs: map[string]*model.Job{
			"test": {
				Name: "test",
				Steps: []*model.Step{
					{Task: ":nonexistent"}, // Reference that doesn't exist
				},
			},
		},
	}

	allPipelines := []*model.Pipeline{mainPipeline, goSkill}

	linter := NewLinterWithPipelines(goSkill, allPipelines)
	errors := linter.Lint()
	assert.Len(t, errors, 1)
	assert.Equal(t, "missing task reference", errors[0].Issue)
	assert.Contains(t, errors[0].Detail, ":nonexistent")
}
