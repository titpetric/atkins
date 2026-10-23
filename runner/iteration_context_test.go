package runner_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
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

			// Verify the iteration variables
			for i, iter := range iterations {
				assert.NotNil(t, iter.Variables, "Iteration %d has nil variables", i)

				// For simple pattern, check that the loop variable exists
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

func TestExpandForDirectVariablePropagatesLazyErrors(t *testing.T) {
	for _, source := range []string{"items", "${items}", "${{ items }}"} {
		t.Run(source, func(t *testing.T) {
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariablesWithResolver(
					map[string]any{"items": "bad"},
					func(string) (string, error) { return "", assert.AnError },
				),
				Step: &model.Step{For: model.Iterators{model.Iterator("item in " + source)}},
				Env:  make(map[string]string),
			}

			_, err := runner.ExpandFor(ctx, nil)
			assert.ErrorIs(t, err, assert.AnError)
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

		// Job variables merge into context BEFORE step execution
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

		// Verify the copy carries the variables
		assert.Equal(t, testBinariesValue, stepCtx.Variables.Get("testBinaries"), "stepCtx should have testBinaries after Copy()")

		// Step 3: Call ExpandFor - it should find testBinaries
		iterations, err := runner.ExpandFor(stepCtx, func(cmd string) (string, error) {
			return "", nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(iterations), "Should have 3 iterations")
	})
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
			step := &model.Step{For: model.Iterators{model.Iterator(tt.forSpec)}}
			ctx := &runner.ExecutionContext{
				Variables: runner.NewContextVariables(tt.vars),
				Env:       make(map[string]string),
				Step:      step,
			}

			iterations, err := runner.ExpandFor(ctx, nil)
			assert.NoError(t, err)
			assert.Len(t, iterations, tt.wantCount)

			for i, expectedVars := range tt.wantVars {
				for key, expectedVal := range expectedVars {
					gotVal := iterations[i].Variables.Get(key)
					assert.NotNil(t, gotVal, "iteration[%d] missing variable %q", i, key)
					assert.Equal(t, expectedVal, gotVal, "iteration[%d].%s", i, key)
				}
			}
		})
	}
}

// BenchmarkExpandForLoop benchmarks for loop expansion
func BenchmarkExpandForLoop(b *testing.B) {
	step := &model.Step{For: model.Iterators{"(i, item) in items"}}
	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{
			"items": []any{"a", "b", "c", "d", "e"},
		}),
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

func TestIterationContext(t *testing.T) {
	t.Parallel()

	ctx := &runner.ExecutionContext{
		Variables: runner.NewContextVariables(map[string]any{"fruits": []any{"apple", "pear"}}),
		Step:      &model.Step{For: model.Iterators{model.Iterator("fruit in fruits")}},
		Env:       make(map[string]string),
	}

	var iterations []runner.IterationContext
	iterations, err := runner.ExpandFor(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(iterations))
	assert.Equal(t, "apple", iterations[0].Variables.Get("fruit"))
	assert.Equal(t, "pear", iterations[1].Variables.Get("fruit"))
	assert.Equal(t, nil, ctx.Variables.Get("fruit"), "each iteration holds its own variables")
}
