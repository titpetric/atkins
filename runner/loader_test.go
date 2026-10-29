package runner_test

import (
	"os"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestLoadPipeline_WithIfConditions tests loading a pipeline with if conditions
func TestLoadPipeline_WithIfConditions(t *testing.T) {
	yamlContent := `
name: If Conditions Test
jobs:
  test:
    steps:
      - run: echo "Always runs"
      - run: echo "Conditional"
        if: "true"
      - run: echo "Conditional false"
        if: "false"
`

	tmpFile := createTempYaml(t, yamlContent)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile))
	})

	pipelines, err := runner.LoadPipeline(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)

	pipeline := pipelines[0]
	assert.Equal(t, "If Conditions Test", pipeline.Name)

	testJob := pipeline.Jobs["test"]
	assert.NotNil(t, testJob)
	assert.Len(t, testJob.Steps, 3)
}

// TestLoadPipeline_WithForLoops tests loading a pipeline with for loops
// Note: The current loader expands for loops at load time using the old yamlexpr approach
// We're keeping this test to verify the loader still works
func TestLoadPipeline_WithForLoops(t *testing.T) {
	yamlContent := `
name: For Loops Test
jobs:
  test:
    vars:
      files:
        - a.txt
        - b.txt
        - c.txt
    steps:
      - run: echo "Processing ${{ item }}"
        for: item in files
`

	tmpFile := createTempYaml(t, yamlContent)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile))
	})

	pipelines, err := runner.LoadPipeline(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)

	pipeline := pipelines[0]
	testJob := pipeline.Jobs["test"]
	assert.NotNil(t, testJob)

	// The current loader expands for loops. We should have 1 step
	assert.Len(t, testJob.Steps, 1)
}

// TestLoadPipeline_WithForLoopsIndexPattern tests loading with (index, item) pattern
// Note: The old loader doesn't support (idx, item) syntax. This tests the new Step.ExpandFor()
func TestLoadPipeline_WithForLoopsIndexPattern(t *testing.T) {
	yamlContent := `
name: For Index Pattern Test
jobs:
  test:
    vars:
      items:
        - alpha
        - beta
        - gamma
    steps:
      - run: echo "Processing alpha"
        name: test-step
`

	tmpFile := createTempYaml(t, yamlContent)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile))
	})

	pipelines, err := runner.LoadPipeline(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)

	pipeline := pipelines[0]
	testJob := pipeline.Jobs["test"]
	assert.NotNil(t, testJob)

	// Test the new Step.ExpandFor() method with index pattern
	step := &model.Step{For: model.Iterators{"(idx, item) in items"}}
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"alpha", "beta", "gamma"},
		}),
		Step: step,
		Env:  make(map[string]string),
	}

	iterations, err := runner.ExpandFor(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, iterations, 3)
}

// createTempYaml creates a temporary YAML file for testing
func createTempYaml(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-*.yml")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, tmpFile.Close())
	})

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)

	return tmpFile.Name()
}

// TestLoadPipeline_JobVariablesDecl tests that job variables are properly loaded into Decl
func TestLoadPipeline_JobVariablesDecl(t *testing.T) {
	yamlContent := `
name: Job Variables Test
jobs:
  test:run:
    vars:
      testBinaries: "file1.test\nfile2.test"
    steps:
      - for: item in testBinaries
        task: test:detail
`

	tmpFile := createTempYaml(t, yamlContent)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tmpFile))
	})

	pipelines, err := runner.LoadPipeline(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)

	pipeline := pipelines[0]
	testJob := pipeline.Jobs["test:run"]
	assert.NotNil(t, testJob)

	// Check that Decl is not nil
	assert.NotNil(t, testJob.Decl, "Job.Decl should not be nil")

	// Check that the loader reads Vars
	assert.NotNil(t, testJob.Vars, "Job.Vars should not be nil")
	assert.NotNil(t, testJob.Vars["testBinaries"], "testBinaries should be in Vars")
	assert.Equal(t, "file1.test\nfile2.test", testJob.Vars["testBinaries"])

	// Now test that MergeVariables properly merges these into the ExecutionContext
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(nil),
		Env:       make(map[string]string),
		Job:       testJob,
	}

	err = runner.MergeVariables(ctx, testJob.Decl)
	assert.NoError(t, err)
	assert.NotNil(t, ctx.Variables.Get("testBinaries"), "testBinaries should be in context after MergeVariables")
	assert.Equal(t, "file1.test\nfile2.test", ctx.Variables.Get("testBinaries"))
}
