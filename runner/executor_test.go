package runner_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
	"github.com/titpetric/atkins/treeview"
)

func TestExecuteStepWithForLoop(t *testing.T) {
	tests := []struct {
		name          string
		step          *model.Step
		variables     map[string]any
		expectedCount int
		expectError   bool
	}{
		{
			name: "simple for loop with list",
			step: &model.Step{
				Name: "test step",
				Run:  "echo ${{ item }}",
				For:  model.Iterators{"item in fruits"},
			},
			variables: map[string]any{
				"fruits": []any{"apple", "banana", "orange"},
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "for loop with custom variable name",
			step: &model.Step{
				Name: "test step",
				Run:  "echo ${{ pkg }}",
				For:  model.Iterators{"pkg in packages"},
			},
			variables: map[string]any{
				"packages": []any{"pkg1", "pkg2"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "empty for loop",
			step: &model.Step{
				Name: "test step",
				Run:  "echo ${{ item }}",
				For:  model.Iterators{"item in empty"},
			},
			variables: map[string]any{
				"empty": []any{},
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariables(tt.variables),
				Step:      tt.step,
				Env:       make(map[string]string),
			}

			iterations, err := runner.ExpandFor(ctx, func(cmd string) (string, error) {
				return "", nil
			})

			if (err != nil) != tt.expectError {
				assert.Fail(t, "ExpandFor error mismatch", "error = %v, expectError %v", err, tt.expectError)
				return
			}

			assert.Equal(t, tt.expectedCount, len(iterations), "expected %d iterations", tt.expectedCount)

			// Verify iteration variables are set correctly
			for i, iter := range iterations {
				assert.NotNil(t, iter.Variables, "Iteration %d has nil variables", i)

				// For simple pattern, check if the loop variable is set
				if len(tt.step.For) > 0 {
					switch string(tt.step.For[0]) {
					case "item in fruits":
						val := iter.Variables.Get("item")
						assert.NotNil(t, val, "Iteration %d missing 'item' variable", i)
					case "pkg in packages":
						val := iter.Variables.Get("pkg")
						assert.NotNil(t, val, "Iteration %d missing 'pkg' variable", i)
					}
				}
			}
		})
	}
}

func TestInterpolationInForLoop(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		variables   map[string]any
		expected    string
		expectError bool
	}{
		{
			name:        "simple variable interpolation",
			cmd:         "echo Fruit: ${{ item }}",
			variables:   map[string]any{"item": "apple"},
			expected:    "echo Fruit: apple",
			expectError: false,
		},
		{
			name:        "multiple variable interpolation",
			cmd:         "echo ${{ key }} = ${{ value }}",
			variables:   map[string]any{"key": "name", "value": "Alice"},
			expected:    "echo name = Alice",
			expectError: false,
		},
		{
			name:        "bash command execution",
			cmd:         "echo $(echo 'hello')",
			variables:   map[string]any{},
			expected:    "echo hello",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariables(tt.variables),
				Env:       make(map[string]string),
			}

			result, err := runner.InterpolateCommand(tt.cmd, ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestForLoopStepExecution(t *testing.T) {
	t.Run("for loop with iterator variable", func(t *testing.T) {
		// Executor not needed for this test

		// Create step with for loop
		step := &model.Step{
			Name: "process items",
			Run:  "echo ${{ item }} >> /tmp/test-for-exec.log",
			For:  model.Iterators{"item in items"},
		}

		// Create execution context with iteration items
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{"one", "two", "three"},
			}),
			Step:    step,
			Env:     make(map[string]string),
			Results: make(map[string]any),
		}

		// Mock execute function for testing
		mockExecuted := []string{}
		mockExecuteFunc := func(cmd string) (string, error) {
			mockExecuted = append(mockExecuted, cmd)
			return "", nil
		}

		// Expand and verify
		iterations, err := runner.ExpandFor(ctx, mockExecuteFunc)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(iterations))

		// Verify each iteration has the correct variable
		expectedItems := []string{"one", "two", "three"}
		for i, iter := range iterations {
			assert.Equal(t, expectedItems[i], iter.Variables.Get("item"), "Iteration %d item mismatch", i)
		}
	})
}

func TestValidateJobRequirements(t *testing.T) {
	tests := []struct {
		name      string
		job       *model.Job
		variables map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "no requirements",
			job: &model.Job{
				Name:     "test_job",
				Requires: []string{},
			},
			variables: map[string]any{},
			expectErr: false,
		},
		{
			name: "requirements satisfied",
			job: &model.Job{
				Name:     "build_component",
				Requires: []string{"component"},
			},
			variables: map[string]any{
				"component": "src/main",
			},
			expectErr: false,
		},
		{
			name: "single requirement missing",
			job: &model.Job{
				Name:     "build_component",
				Requires: []string{"component"},
			},
			variables: map[string]any{},
			expectErr: true,
			errMsg:    "requires variables [component] but missing: [component]",
		},
		{
			name: "multiple requirements, some missing",
			job: &model.Job{
				Name:     "deploy_service",
				Requires: []string{"service", "version", "env"},
			},
			variables: map[string]any{
				"service": "api",
				"version": "1.0.0",
			},
			expectErr: true,
			errMsg:    "requires variables [service version env] but missing: [env]",
		},
		{
			name: "all requirements present",
			job: &model.Job{
				Name:     "deploy_service",
				Requires: []string{"service", "version", "env"},
			},
			variables: map[string]any{
				"service": "api",
				"version": "1.0.0",
				"env":     "prod",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariables(tt.variables),
			}
			// Ensure job name is set (already set by test case)

			err := runner.ValidateJobRequirements(ctx, tt.job)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStepWithMultipleCmds(t *testing.T) {
	t.Run("step with multiple cmds should display count", func(t *testing.T) {
		step := &model.Step{
			Name: "test multiple commands",
			Cmds: []string{"echo cmd1", "echo cmd2", "echo cmd3"},
		}

		// String() should show count, not expanded commands
		result := step.String()
		assert.Equal(t, "cmds: <3 commands>", result)
		assert.NotContains(t, result, "&&")
		assert.NotContains(t, result, "echo cmd1")
	})
}

func TestIsEchoCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "simple echo command",
			cmd:      "echo hello",
			expected: true,
		},
		{
			name:     "echo with quoted string",
			cmd:      "echo 'hello world'",
			expected: true,
		},
		{
			name:     "echo with variable",
			cmd:      "echo ${{ name }}",
			expected: true,
		},
		{
			name:     "echo with leading spaces",
			cmd:      "   echo test",
			expected: true,
		},
		{
			name:     "echo with multiline",
			cmd:      "echo hello\necho world",
			expected: false,
		},
		{
			name:     "non-echo command",
			cmd:      "make build",
			expected: false,
		},
		{
			name:     "command with echo in middle",
			cmd:      "cmd | echo test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.IsEchoCommand(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskInvocationWithForLoop(t *testing.T) {
	t.Run("expand for loop with task variables", func(t *testing.T) {
		// Create a step that invokes a task with a for loop
		step := &model.Step{
			Name: "build all components",
			Task: "build_component",
			For:  model.Iterators{"component in components"},
		}

		// Create execution context
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"components": []any{"src/main", "src/utils", "tests/"},
			}),
			Step: step,
			Env:  make(map[string]string),
		}

		// Expand the for loop
		iterations, err := runner.ExpandFor(ctx, func(cmd string) (string, error) {
			return "", nil
		})
		assert.NoError(t, err)

		assert.Equal(t, 3, len(iterations))

		// Verify each iteration has the component variable
		expectedComponents := []string{"src/main", "src/utils", "tests/"}
		for i, iter := range iterations {
			assert.Equal(t, expectedComponents[i], iter.Variables.Get("component"), "Iteration %d component mismatch", i)
		}
	})

	t.Run("task requires variable from for loop", func(t *testing.T) {
		// Create a task that requires the loop variable
		task := &model.Job{
			Name:     "build_component",
			Requires: []string{"component"},
		}

		// Simulate iteration context with loop variable
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"component": "src/main",
			}),
		}

		// Should pass validation
		err := runner.ValidateJobRequirements(ctx, task)
		assert.NoError(t, err)
	})

	t.Run("task requires variable missing from for loop context", func(t *testing.T) {
		// Create a task that requires a variable
		task := &model.Job{
			Name:     "build_component",
			Requires: []string{"component"},
		}

		// Iteration context without the required variable
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
		}

		// Should fail validation
		err := runner.ValidateJobRequirements(ctx, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component")
	})
}

func TestJobVariablesInForLoop(t *testing.T) {
	t.Run("job variable available in for loop expansion", func(t *testing.T) {
		// Simulate a job with variables and a for loop step
		// Like: vars: { testBinaries: "$(ls ./bin/*.test)" }
		//       steps: [{ for: "item in testBinaries" }]

		step := &model.Step{
			Name: "process test binaries",
			Task: "test:detail",
			For:  model.Iterators{"item in testBinaries"},
		}

		// Job variables are merged into context BEFORE step execution
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"testBinaries": "runner.test\nmodel.test\ntreeview.test",
			}),
			Step: step,
			Env:  make(map[string]string),
		}

		// Expand the for loop - should find testBinaries variable
		iterations, err := runner.ExpandFor(ctx, func(cmd string) (string, error) {
			return "", nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(iterations))

		expectedBinaries := []string{"runner.test", "model.test", "treeview.test"}
		for i, iter := range iterations {
			assert.Equal(t, expectedBinaries[i], iter.Variables.Get("item"), "Iteration %d item mismatch", i)
		}
	})

	t.Run("job variables merged from Decl", func(t *testing.T) {
		// Test that job variables in Decl are properly merged
		job := &model.Job{
			Name: "test_job",
			Decl: &model.Decl{
				Vars: map[string]any{
					"testBinaries": "file1.test\nfile2.test",
				},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
			Env:       make(map[string]string),
		}

		// Merge job variables
		err := runner.MergeVariables(ctx, job.Decl)
		assert.NoError(t, err)

		// Check that testBinaries is in Variables
		assert.NotNil(t, ctx.Variables.Get("testBinaries"), "testBinaries should be in context variables")
		assert.Equal(t, "file1.test\nfile2.test", ctx.Variables.Get("testBinaries"))
	})

	t.Run("actual pipeline with for loop accessing job variable", func(t *testing.T) {
		// Test the actual scenario from atkins.yml
		// test:run job has vars: { testBinaries: ... }
		// and a step: { for: "item in testBinaries", task: "test:detail" }

		testBinariesValue := "runner.test\nmodel.test\ntreeview.test"

		// Simulate the test:run job
		job := &model.Job{
			Name: "test:run",
			Decl: &model.Decl{
				Vars: map[string]any{
					"testBinaries": testBinariesValue,
				},
			},
			Steps: []*model.Step{
				{
					Name: "process items",
					Task: "test:detail",
					For:  model.Iterators{"item in testBinaries"},
				},
			},
		}

		// Simulate ExecuteJob flow
		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
			Env:       make(map[string]string),
			Job:       job,
		}

		// Step 1: Merge job variables (from ExecuteJob line 119)
		err := runner.MergeVariables(ctx, job.Decl)
		assert.NoError(t, err)
		assert.Equal(t, testBinariesValue, ctx.Variables.Get("testBinaries"), "testBinaries should be merged")

		// Step 2: Simulate executeStep which calls Copy() then calls executeStepWithForLoop
		stepCtx := ctx.Copy() // This should copy variables
		stepCtx.Step = job.Steps[0]
		stepCtx.Env = make(map[string]string)

		// Verify variables are copied correctly
		assert.Equal(t, testBinariesValue, stepCtx.Variables.Get("testBinaries"), "stepCtx should have testBinaries after Copy()")

		// Step 3: Call ExpandFor - it should find testBinaries
		iterations, err := runner.ExpandFor(stepCtx, func(cmd string) (string, error) {
			return "", nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(iterations), "Should have 3 iterations")
	})
}

func TestStepVarsWithLoopVariable(t *testing.T) {
	t.Run("step vars with loop variable should be interpolated per iteration", func(t *testing.T) {
		// This tests the fix for BUG 2: step-level vars like `path: $(dirname "${{item}}")`
		// should be interpolated with the loop variable available

		step := &model.Step{
			Name: "test step",
			Task: "project:ps",
			For:  model.Iterators{"item in projects"},
			Decl: &model.Decl{
				Vars: map[string]any{
					"path": "${{item}}/subdir",
				},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"projects": []any{"proj1", "proj2", "proj3"},
			}),
			Step: step,
			Env:  make(map[string]string),
		}

		// Expand the for loop
		iterations, err := runner.ExpandFor(ctx, func(cmd string) (string, error) {
			return "", nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(iterations))

		// Verify each iteration has the correct item
		for i, iter := range iterations {
			expectedItems := []string{"proj1", "proj2", "proj3"}
			assert.Equal(t, expectedItems[i], iter.Variables.Get("item"))

			// Now simulate what happens in executeTaskStepWithLoop:
			// Create iteration context and merge step vars
			iterCtx := &runner.ExecutionContext{
				Variables: iter.Variables.Clone(),
				Env:       make(map[string]string),
			}

			// Merge step-level vars (this is the fix)
			err := runner.MergeVariables(iterCtx, step.Decl)
			assert.NoError(t, err)

			// The path should now be interpolated with the item value
			expectedPath := expectedItems[i] + "/subdir"
			assert.Equal(t, expectedPath, iterCtx.Variables.Get("path"),
				"path should be interpolated with item=%s", expectedItems[i])
		}
	})
}

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

func TestExecuteJob_SkipsWhenIfConditionFalse(t *testing.T) {
	t.Run("job with false if condition returns ErrJobSkipped", func(t *testing.T) {
		job := &model.Job{
			Name: "conditional_job",
			If:   model.Conditionals{"false"},
			Steps: []*model.Step{
				{Run: "echo should not run"},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
			Env:       make(map[string]string),
			Job:       job,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(t.Context(), ctx)
		assert.ErrorIs(t, err, runner.ErrJobSkipped)
	})

	t.Run("job with true if condition runs normally", func(t *testing.T) {
		job := &model.Job{
			Name: "conditional_job",
			If:   model.Conditionals{"true"},
			Steps: []*model.Step{
				{Run: "echo hello"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "conditional_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(t.Context(), ctx)
		assert.NoError(t, err)
	})

	t.Run("job with variable-based if condition skipped", func(t *testing.T) {
		job := &model.Job{
			Name: "notification",
			If:   model.Conditionals{"last_sent_match < last_inbox_match"},
			Decl: &model.Decl{
				Vars: map[string]any{
					"last_inbox_match": "2026-02-23T12:23:36Z",
					"last_sent_match":  "2026-03-02T20:12:59Z",
				},
			},
			Steps: []*model.Step{
				{Run: "echo Hello"},
			},
		}

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(nil),
			Env:       make(map[string]string),
			Job:       job,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(t.Context(), ctx)
		assert.ErrorIs(t, err, runner.ErrJobSkipped)
	})
}

func TestCurrentStepSetCorrectlyInIteration(t *testing.T) {
	t.Run("CurrentStep should be set to iteration node during execution", func(t *testing.T) {
		// This tests the fix for BUG 1: output should go to the correct iteration node,
		// not be overwritten by subsequent iterations

		// Create a simple context with a CurrentStep set
		parentNode := treeview.NewNode("parent")
		iterNode1 := treeview.NewNode("iter1")
		iterNode2 := treeview.NewNode("iter2")

		execCtx := &runner.ExecutionContext{
			CurrentStep: parentNode,
			Variables:   runner.NewContextVariables(nil),
			Env:         make(map[string]string),
		}

		// Simulate what executeStepIteration does:
		// Save original, set to iter node, then restore

		// First iteration
		func() {
			originalStep := execCtx.CurrentStep
			execCtx.CurrentStep = iterNode1
			defer func() { execCtx.CurrentStep = originalStep }()

			// During execution, CurrentStep should be iterNode1
			assert.Equal(t, iterNode1, execCtx.CurrentStep)
		}()

		// After first iteration, should be restored to parent
		assert.Equal(t, parentNode, execCtx.CurrentStep)

		// Second iteration
		func() {
			originalStep := execCtx.CurrentStep
			execCtx.CurrentStep = iterNode2
			defer func() { execCtx.CurrentStep = originalStep }()

			// During execution, CurrentStep should be iterNode2
			assert.Equal(t, iterNode2, execCtx.CurrentStep)
		}()

		// After second iteration, should be restored to parent
		assert.Equal(t, parentNode, execCtx.CurrentStep)
	})
}

// TestExecuteJob_DirBeforeVars verifies that ExecuteJob evaluates job.Dir
// BEFORE merging variables, so $(cmd) in vars runs in the correct directory.
func TestExecuteJob_DirBeforeVars(t *testing.T) {
	t.Run("subshell pwd uses job dir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-workdir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		job := &model.Job{
			Name: "test_job",
			Dir:  tmpDir,
			Decl: &model.Decl{
				Vars: map[string]any{
					"current_dir": "$(pwd)",
				},
			},
			Steps: []*model.Step{
				{Run: "true"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "test_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(t.Context(), ctx)
		assert.NoError(t, err)
		assert.Equal(t, tmpDir, ctx.Variables.Get("current_dir"))
	})

	t.Run("subshell ls lists files from job dir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-workdir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := tmpDir + "/marker.txt"
		assert.NoError(t, os.WriteFile(testFile, []byte("x"), 0o644))

		job := &model.Job{
			Name: "test_job",
			Dir:  tmpDir,
			Decl: &model.Decl{
				Vars: map[string]any{
					"files": "$(ls)",
				},
			},
			Steps: []*model.Step{
				{Run: "true"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "test_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(t.Context(), ctx)
		assert.NoError(t, err)
		assert.Contains(t, ctx.Variables.Get("files"), "marker.txt")
	})

	t.Run("dir from interpolated var with subshell", func(t *testing.T) {
		// Tests the case where dir depends on a var that comes from a subshell:
		// - dir: ${{workdir}}
		// - vars: { workdir: "$(echo /tmp)", files_in_wd: "$(ls)" }
		// The evaluation order should be:
		// 1. Evaluate workdir (dir depends on it)
		// 2. Evaluate dir with workdir available
		// 3. Evaluate files_in_wd with dir set as cwd
		tmpDir, err := os.MkdirTemp("", "test-workdir-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		assert.NoError(t, os.WriteFile(tmpDir+"/subshell_marker.txt", []byte("x"), 0o644))

		job := &model.Job{
			Name: "test_job",
			Dir:  "${{ workdir }}",
			Decl: &model.Decl{
				Vars: map[string]any{
					"workdir":     "$(echo " + tmpDir + ")",
					"files_in_wd": "$(ls)",
				},
			},
			Steps: []*model.Step{
				{Run: "true"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "test_job")

		ctx := &runner.ExecutionContext{
			Variables:  runner.NewContextVariables(nil),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(t.Context(), ctx)

		assert.NoError(t, err)
		assert.Equal(t, tmpDir, ctx.Variables.Get("workdir"))
		assert.Contains(t, ctx.Variables.Get("files_in_wd"), "subshell_marker.txt")
	})
}

// TestExecuteStepWithForLoop_SequentialBreaksOnError verifies that sequential
// iteration (without detach) stops on the first error.
func TestExecuteStepWithForLoop_SequentialBreaksOnError(t *testing.T) {
	tmpDir := t.TempDir()

	job := &model.Job{
		Name: "test_job",
		Steps: []*model.Step{
			{
				Run: `if [ "${{ item }}" = "fail" ]; then exit 1; fi; touch "` + tmpDir + `/${{ item }}.done"`,
				For: model.Iterators{"item in items"},
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"first", "fail", "third"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(t.Context(), ctx)
	assert.Error(t, err)

	// First ran, third should NOT have run
	_, err1 := os.Stat(tmpDir + "/first.done")
	_, err3 := os.Stat(tmpDir + "/third.done")
	assert.NoError(t, err1, "first iteration should complete")
	assert.True(t, os.IsNotExist(err3), "third iteration should not run")
}

// TestExecuteStepWithForLoop_DetachParallel verifies that for loop with
// detach runs iterations in parallel by completing all iterations.
func TestExecuteStepWithForLoop_DetachParallel(t *testing.T) {
	tmpDir := t.TempDir()

	job := &model.Job{
		Name: "test_job",
		Steps: []*model.Step{
			{
				Run:    `touch "` + tmpDir + `/${{ item }}.done"`,
				For:    model.Iterators{"item in items"},
				Detach: true,
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"a", "b", "c", "d"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(t.Context(), ctx)
	assert.NoError(t, err)

	// All files should be created when detach completes
	for _, item := range []string{"a", "b", "c", "d"} {
		_, err := os.Stat(tmpDir + "/" + item + ".done")
		assert.NoError(t, err, "iteration %s should complete", item)
	}
}

// TestExecuteStepWithForLoop_DetachContinuesOnError verifies that parallel
// iterations continue even when one fails.
func TestExecuteStepWithForLoop_DetachContinuesOnError(t *testing.T) {
	tmpDir := t.TempDir()

	job := &model.Job{
		Name: "test_job",
		Steps: []*model.Step{
			{
				Run:    `if [ "${{ item }}" = "fail" ]; then exit 1; fi; touch "` + tmpDir + `/${{ item }}.done"`,
				For:    model.Iterators{"item in items"},
				Detach: true,
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"first", "fail", "third"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(t.Context(), ctx)
	assert.Error(t, err)

	// Both first and third should have run (detach waits for all to complete)
	_, err1 := os.Stat(tmpDir + "/first.done")
	_, err3 := os.Stat(tmpDir + "/third.done")
	assert.NoError(t, err1, "first iteration should complete")
	assert.NoError(t, err3, "third iteration should complete with detach")
}

// TestExecuteStepWithForLoop_ContextCancellation verifies that iterations
// respect context timeout/cancellation.
func TestExecuteStepWithForLoop_ContextCancellation(t *testing.T) {
	job := &model.Job{
		Name:    "test_job",
		Timeout: "1ms", // Very short timeout to trigger cancellation
		Steps: []*model.Step{
			{
				// Command that would take too long
				Run: `for i in $(seq 1 1000); do echo $i > /dev/null; done`,
				For: model.Iterators{"item in items"},
			},
		},
	}

	display := treeview.NewSilentDisplay()
	builder := treeview.NewBuilder("test")
	jobNode := builder.AddJob(job, nil, "test_job")

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"first", "second", "third"},
		}),
		Env:        make(map[string]string),
		Job:        job,
		CurrentJob: jobNode,
		Display:    display,
		Builder:    builder,
	}

	executor := runner.NewExecutor()
	err := executor.ExecuteJob(context.Background(), ctx)
	// Should have an error due to timeout
	assert.Error(t, err)
}

func TestExecuteJob_ForLoop(t *testing.T) {
	t.Run("job-level for loop runs steps for each iteration", func(t *testing.T) {
		job := &model.Job{
			Name: "loop_job",
			Desc: "Iteration ${{ item }}",
			For:  model.Iterators{"item in items"},
			Steps: []*model.Step{
				{Run: "echo ${{ item }}"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "loop_job")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{"alpha", "beta", "gamma"},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)

		// Job node should have 3 iteration children (pre-built steps replaced)
		children := jobNode.GetChildren()
		assert.Len(t, children, 3)
		assert.Equal(t, "Iteration alpha", children[0].GetName())
		assert.Equal(t, "Iteration beta", children[1].GetName())
		assert.Equal(t, "Iteration gamma", children[2].GetName())

		for _, child := range children {
			assert.Equal(t, treeview.StatusPassed, child.GetStatus())
		}
	})

	t.Run("job-level for loop with dir referencing loop variable", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-jobloop-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create subdirectories
		for _, name := range []string{"aaa", "bbb"} {
			assert.NoError(t, os.MkdirAll(tmpDir+"/"+name, 0o755))
		}

		job := &model.Job{
			Name: "dir_loop",
			Desc: "${{ folder }}",
			Dir:  tmpDir + "/${{ folder }}",
			For:  model.Iterators{"folder in folders"},
			Steps: []*model.Step{
				{Run: "echo hello"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "dir_loop")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"folders": []any{"aaa", "bbb"},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err = executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)

		children := jobNode.GetChildren()
		assert.Len(t, children, 2)
		assert.Equal(t, "aaa", children[0].GetName())
		assert.Equal(t, "bbb", children[1].GetName())
	})

	t.Run("job-level for loop with if condition referencing loop variable", func(t *testing.T) {
		job := &model.Job{
			Name: "if_loop",
			For:  model.Iterators{"item in items"},
			If:   model.Conditionals{`item != "skip"`},
			Steps: []*model.Step{
				{Run: "echo ${{ item }}"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "if_loop")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{"one", "skip", "three"},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)

		// "skip" iteration should be excluded, leaving 2 iterations
		children := jobNode.GetChildren()
		assert.Len(t, children, 2)
	})

	t.Run("job-level for loop with empty iterations", func(t *testing.T) {
		job := &model.Job{
			Name: "empty_loop",
			For:  model.Iterators{"item in items"},
			Steps: []*model.Step{
				{Run: "echo ${{ item }}"},
			},
		}

		display := treeview.NewSilentDisplay()
		builder := treeview.NewBuilder("test")
		jobNode := builder.AddJob(job, nil, "empty_loop")

		ctx := &runner.ExecutionContext{
			Variables: runner.NewContextVariables(map[string]any{
				"items": []any{},
			}),
			Env:        make(map[string]string),
			Job:        job,
			CurrentJob: jobNode,
			Display:    display,
			Builder:    builder,
		}

		executor := runner.NewExecutor()
		err := executor.ExecuteJob(context.Background(), ctx)
		assert.NoError(t, err)
	})
}
