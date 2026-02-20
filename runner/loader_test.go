package runner_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// The current loader expands for loops, so we should have 1 step
	assert.Len(t, testJob.Steps, 1)
}

// TestLoadPipeline_WithForLoopsIndexPattern tests loading with (index, item) pattern
// Note: The old loader doesn't support (idx, item) syntax, so this tests the new Step.ExpandFor()
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
	step := &model.Step{For: "(idx, item) in items"}
	ctx := &runner.ExecutionContext{
		Variables: map[string]any{
			"items": []any{"alpha", "beta", "gamma"},
		},
		Step: step,
		Env:  make(map[string]string),
	}

	iterations, err := runner.ExpandFor(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, iterations, 3)
}

// TestEvaluateIfInContext tests if conditions with context variables
func TestEvaluateIfInContext(t *testing.T) {
	tests := []struct {
		name     string
		ifCond   string
		vars     map[string]any
		env      map[string]string
		wantBool bool
	}{
		{
			name:     "context variable comparison",
			ifCond:   "matrix_os == 'linux'",
			vars:     map[string]any{"matrix_os": "linux"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "env variable check",
			ifCond:   "GOARCH == 'amd64'",
			vars:     make(map[string]any),
			env:      map[string]string{"GOARCH": "amd64"},
			wantBool: true,
		},
		{
			name:     "combined condition",
			ifCond:   "matrix_os == 'linux' && GOARCH == 'amd64'",
			vars:     map[string]any{"matrix_os": "linux"},
			env:      map[string]string{"GOARCH": "amd64"},
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &model.Step{If: tt.ifCond}
			ctx := &runner.ExecutionContext{
				Variables: tt.vars,
				Env:       tt.env,
				Step:      step,
			}

			result, err := runner.EvaluateIf(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantBool, result)
		})
	}
}

// TestExpandForWithVariables tests expanding for loops with context variables
func TestExpandForWithVariables(t *testing.T) {
	tests := []struct {
		name      string
		forSpec   string
		vars      map[string]any
		wantCount int
		wantVars  map[int]map[string]any
	}{
		{
			name:      "simple item pattern",
			forSpec:   "item in targets",
			vars:      map[string]any{"targets": []any{"test1", "test2"}},
			wantCount: 2,
			wantVars: map[int]map[string]any{
				0: {"item": "test1"},
				1: {"item": "test2"},
			},
		},
		{
			name:      "index, item pattern",
			forSpec:   "(i, item) in targets",
			vars:      map[string]any{"targets": []any{"test1", "test2"}},
			wantCount: 2,
			wantVars: map[int]map[string]any{
				0: {"i": 0, "item": "test1"},
				1: {"i": 1, "item": "test2"},
			},
		},
		{
			name:      "inline array literal",
			forSpec:   `test in ["detach", "depends_on", "root_jobs", "nested"]`,
			vars:      map[string]any{},
			wantCount: 4,
			wantVars: map[int]map[string]any{
				0: {"test": "detach"},
				1: {"test": "depends_on"},
				2: {"test": "root_jobs"},
				3: {"test": "nested"},
			},
		},
		{
			name:      "inline integer array literal",
			forSpec:   `num in [1, 2, 3]`,
			vars:      map[string]any{},
			wantCount: 3,
			wantVars: map[int]map[string]any{
				0: {"num": 1},
				1: {"num": 2},
				2: {"num": 3},
			},
		},
		{
			name:      "inline mixed array literal",
			forSpec:   `item in ["hello", 42, "world"]`,
			vars:      map[string]any{},
			wantCount: 3,
			wantVars: map[int]map[string]any{
				0: {"item": "hello"},
				1: {"item": 42},
				2: {"item": "world"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &model.Step{For: tt.forSpec}
			ctx := &runner.ExecutionContext{
				Variables: tt.vars,
				Env:       make(map[string]string),
				Step:      step,
			}

			iterations, err := runner.ExpandFor(ctx, nil)
			assert.NoError(t, err)
			assert.Len(t, iterations, tt.wantCount)

			for i, expectedVars := range tt.wantVars {
				for key, expectedVal := range expectedVars {
					gotVal, ok := iterations[i].Variables[key]
					assert.True(t, ok, "iteration[%d] missing variable %q", i, key)
					assert.Equal(t, expectedVal, gotVal, "iteration[%d].%s", i, key)
				}
			}
		})
	}
}

// createTempYaml creates a temporary YAML file for testing
func createTempYaml(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-*.yml")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, tmpFile.Close())
	})

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	return tmpFile.Name()
}

// BenchmarkEvaluateIfExpression benchmarks if condition evaluation
func BenchmarkEvaluateIfExpression(b *testing.B) {
	step := &model.Step{If: "matrix_os == 'linux' && GOARCH == 'amd64'"}
	ctx := &runner.ExecutionContext{
		Variables: map[string]any{"matrix_os": "linux"},
		Env:       map[string]string{"GOARCH": "amd64"},
		Step:      step,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.EvaluateIf(ctx)
		if err != nil {
			b.Fatalf("EvaluateIf failed: %v", err)
		}
	}
}

// BenchmarkExpandForLoop benchmarks for loop expansion
func BenchmarkExpandForLoop(b *testing.B) {
	step := &model.Step{For: "(i, item) in items"}
	ctx := &runner.ExecutionContext{
		Variables: map[string]any{
			"items": []any{"a", "b", "c", "d", "e"},
		},
		Step: step,
		Env:  make(map[string]string),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.ExpandFor(ctx, nil)
		if err != nil {
			b.Fatalf("ExpandFor failed: %v", err)
		}
	}
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

	// Check that Vars are loaded
	assert.NotNil(t, testJob.Vars, "Job.Vars should not be nil")
	assert.NotNil(t, testJob.Vars["testBinaries"], "testBinaries should be in Vars")
	assert.Equal(t, "file1.test\nfile2.test", testJob.Vars["testBinaries"])

	// Now test that MergeVariables properly merges these into the ExecutionContext
	ctx := &runner.ExecutionContext{
		Variables: make(map[string]any),
		Env:       make(map[string]string),
		Job:       testJob,
	}

	err = runner.MergeVariables(ctx, testJob.Decl)
	assert.NoError(t, err)
	assert.NotNil(t, ctx.Variables["testBinaries"], "testBinaries should be in context after MergeVariables")
	assert.Equal(t, "file1.test\nfile2.test", ctx.Variables["testBinaries"])
}
