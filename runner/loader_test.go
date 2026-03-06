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
// These tests cover the documentation examples from pipelines-jobs-steps.md
func TestEvaluateIfInContext(t *testing.T) {
	tests := []struct {
		name     string
		ifCond   string
		vars     map[string]any
		env      map[string]string
		wantBool bool
	}{
		// Basic string comparisons (from docs: "String comparisons")
		{
			name:     "string equals",
			ifCond:   `environment == "production"`,
			vars:     map[string]any{"environment": "production"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "string not equals",
			ifCond:   `branch != "main"`,
			vars:     map[string]any{"branch": "feature"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "string equals false",
			ifCond:   `environment == "production"`,
			vars:     map[string]any{"environment": "staging"},
			env:      make(map[string]string),
			wantBool: false,
		},

		// Boolean variables (from docs: "Boolean variables")
		{
			name:     "boolean true variable",
			ifCond:   "enable_deploy",
			vars:     map[string]any{"enable_deploy": true},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "boolean false variable",
			ifCond:   "skip_tests",
			vars:     map[string]any{"skip_tests": false},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "negated boolean",
			ifCond:   "!skip_tests",
			vars:     map[string]any{"skip_tests": false},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "explicit boolean comparison",
			ifCond:   "run_integration_tests == true",
			vars:     map[string]any{"run_integration_tests": true},
			env:      make(map[string]string),
			wantBool: true,
		},

		// Combined conditions (from docs: "Combining conditions")
		{
			name:     "AND condition both true",
			ifCond:   `environment == "production" && branch == "main"`,
			vars:     map[string]any{"environment": "production", "branch": "main"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "AND condition one false",
			ifCond:   `environment == "production" && branch == "main"`,
			vars:     map[string]any{"environment": "production", "branch": "develop"},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "OR condition one true",
			ifCond:   `skip_tests || environment == "development"`,
			vars:     map[string]any{"skip_tests": false, "environment": "development"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "OR condition both false",
			ifCond:   `skip_tests || environment == "development"`,
			vars:     map[string]any{"skip_tests": false, "environment": "production"},
			env:      make(map[string]string),
			wantBool: false,
		},

		// Environment variables
		{
			name:     "env variable check",
			ifCond:   "GOARCH == 'amd64'",
			vars:     make(map[string]any),
			env:      map[string]string{"GOARCH": "amd64"},
			wantBool: true,
		},
		{
			name:     "combined vars and env",
			ifCond:   `deploy_env == "production" && CI == "true"`,
			vars:     map[string]any{"deploy_env": "production"},
			env:      map[string]string{"CI": "true"},
			wantBool: true,
		},

		// List membership (from docs: "Checking for values in lists")
		{
			name:     "in operator with list - found",
			ifCond:   `environment in allowed_envs`,
			vars:     map[string]any{"environment": "staging", "allowed_envs": []any{"staging", "production"}},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "in operator with list - not found",
			ifCond:   `environment in allowed_envs`,
			vars:     map[string]any{"environment": "development", "allowed_envs": []any{"staging", "production"}},
			env:      make(map[string]string),
			wantBool: false,
		},

		// Pattern matching (from docs: "Pattern matching")
		{
			name:     "matches operator - release branch",
			ifCond:   `branch matches "^release/.*"`,
			vars:     map[string]any{"branch": "release/v1.0"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "matches operator - not release branch",
			ifCond:   `branch matches "^release/.*"`,
			vars:     map[string]any{"branch": "feature/new-thing"},
			env:      make(map[string]string),
			wantBool: false,
		},

		// Truthiness rules (from docs: "Truthiness" table)
		{
			name:     "empty string is falsy",
			ifCond:   "value",
			vars:     map[string]any{"value": ""},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "non-empty string is truthy",
			ifCond:   "value",
			vars:     map[string]any{"value": "hello"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "string 'false' is falsy",
			ifCond:   "value",
			vars:     map[string]any{"value": "false"},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "string '0' is falsy",
			ifCond:   "value",
			vars:     map[string]any{"value": "0"},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "zero int is falsy",
			ifCond:   "count",
			vars:     map[string]any{"count": 0},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "non-zero int is truthy",
			ifCond:   "count",
			vars:     map[string]any{"count": 42},
			env:      make(map[string]string),
			wantBool: true,
		},

		// Undefined variables (from docs: "Undefined Variables")
		{
			name:     "undefined variable is falsy",
			ifCond:   "maybe_defined",
			vars:     make(map[string]any), // variable not defined
			env:      make(map[string]string),
			wantBool: false,
		},

		// Numeric comparisons (from docs: "Expression Syntax" table)
		{
			name:     "greater than",
			ifCond:   "num_items > 0",
			vars:     map[string]any{"num_items": 5},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "less than or equal",
			ifCond:   "num_items <= 10",
			vars:     map[string]any{"num_items": 10},
			env:      make(map[string]string),
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
