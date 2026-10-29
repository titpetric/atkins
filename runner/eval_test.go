package runner_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

func TestEvaluateJobIf(t *testing.T) {
	tests := []struct {
		name     string
		job      *model.Job
		vars     map[string]any
		env      map[string]string
		wantBool bool
		wantErr  bool
	}{
		{
			name:     "no if condition",
			job:      &model.Job{Name: "test"},
			vars:     map[string]any{},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "nil job",
			job:      nil,
			vars:     map[string]any{},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "true condition",
			job:      &model.Job{Name: "test", If: model.Conditionals{"true"}},
			vars:     map[string]any{},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "false condition",
			job:      &model.Job{Name: "test", If: model.Conditionals{"false"}},
			vars:     map[string]any{},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "variable comparison true",
			job:      &model.Job{Name: "test", If: model.Conditionals{"last_sent_match < last_inbox_match"}},
			vars:     map[string]any{"last_sent_match": "2026-02-20T00:00:00Z", "last_inbox_match": "2026-02-23T12:23:36Z"},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "variable comparison false",
			job:      &model.Job{Name: "test", If: model.Conditionals{"last_sent_match < last_inbox_match"}},
			vars:     map[string]any{"last_sent_match": "2026-03-02T20:12:59Z", "last_inbox_match": "2026-02-23T12:23:36Z"},
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "multiple conditions all true",
			job:      &model.Job{Name: "test", If: model.Conditionals{"enabled == true", "num_items > 0"}},
			vars:     map[string]any{"enabled": true, "num_items": 1},
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "multiple conditions one false",
			job:      &model.Job{Name: "test", If: model.Conditionals{"enabled == true", "num_items > 0"}},
			vars:     map[string]any{"enabled": true, "num_items": 0},
			env:      make(map[string]string),
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariables(tt.vars),
				Env:       tt.env,
				Job:       tt.job,
			}

			result, err := runner.EvaluateJobIf(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantBool, result)
		})
	}
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

		// Command exit statuses
		{
			name:     "successful command with no output is truthy",
			ifCond:   "$(test -d .)",
			vars:     make(map[string]any),
			env:      make(map[string]string),
			wantBool: true,
		},
		{
			name:     "failed command with no output is falsy",
			ifCond:   "$(test -e ./path-that-does-not-exist)",
			vars:     make(map[string]any),
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "exit status decides when the output reads true",
			ifCond:   "$(printf true; false)",
			vars:     make(map[string]any),
			env:      make(map[string]string),
			wantBool: false,
		},
		{
			name:     "successful command output remains an expression",
			ifCond:   "$(printf true)",
			vars:     make(map[string]any),
			env:      make(map[string]string),
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &model.Step{If: model.Conditionals{model.Condition(tt.ifCond)}}
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariables(tt.vars),
				Env:       tt.env,
				Step:      step,
			}

			result, err := runner.EvaluateIf(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantBool, result)
		})
	}
}

// BenchmarkEvaluateIfExpression benchmarks if condition evaluation
func BenchmarkEvaluateIfExpression(b *testing.B) {
	step := &model.Step{If: model.Conditionals{"matrix_os == 'linux' && GOARCH == 'amd64'"}}
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{"matrix_os": "linux"}),
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
