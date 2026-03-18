package runner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// TestContextVariables_ImplementsInterface verifies the interface contract.
func TestContextVariables_ImplementsInterface(t *testing.T) {
	var _ model.VariableStorage = runner.NewContextVariables(nil)
}

// TestContextVariables_Get tests basic Get functionality.
func TestContextVariables_Get(t *testing.T) {
	t.Run("returns nil for missing key", func(t *testing.T) {
		vars := runner.NewContextVariables(nil)
		assert.Nil(t, vars.Get("missing"))
	})

	t.Run("returns value for existing key", func(t *testing.T) {
		vars := runner.NewContextVariables(map[string]any{
			"name": "alice",
			"age":  30,
		})
		assert.Equal(t, "alice", vars.Get("name"))
		assert.Equal(t, 30, vars.Get("age"))
	})

	t.Run("returns nil for missing after Set", func(t *testing.T) {
		vars := runner.NewContextVariables(nil)
		vars.Set("key", "value")
		assert.Equal(t, "value", vars.Get("key"))
		assert.Nil(t, vars.Get("other"))
	})
}

// TestContextVariables_Set tests basic Set functionality.
func TestContextVariables_Set(t *testing.T) {
	t.Run("sets new value", func(t *testing.T) {
		vars := runner.NewContextVariables(nil)
		vars.Set("key", "value")
		assert.Equal(t, "value", vars.Get("key"))
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		vars := runner.NewContextVariables(map[string]any{"key": "old"})
		vars.Set("key", "new")
		assert.Equal(t, "new", vars.Get("key"))
	})
}

// TestContextVariables_LazyEvaluation tests lazy evaluation of pending vars.
func TestContextVariables_LazyEvaluation(t *testing.T) {
	t.Run("evaluates pending var on Get", func(t *testing.T) {
		evalCount := 0
		resolver := func(s string) (string, error) {
			evalCount++
			return "resolved:" + s, nil
		}

		vars := runner.NewContextVariablesWithResolver(
			map[string]any{"lazy": "input"},
			resolver,
		)

		assert.Equal(t, 0, evalCount, "should not evaluate before Get")

		result := vars.Get("lazy")
		assert.Equal(t, "resolved:input", result)
		assert.Equal(t, 1, evalCount, "should evaluate on first Get")

		// Second Get should return cached value
		result2 := vars.Get("lazy")
		assert.Equal(t, "resolved:input", result2)
		assert.Equal(t, 1, evalCount, "should not re-evaluate on second Get")
	})

	t.Run("non-string pending values pass through", func(t *testing.T) {
		vars := runner.NewContextVariablesWithResolver(
			map[string]any{"num": 42, "flag": true},
			nil,
		)
		assert.Equal(t, 42, vars.Get("num"))
		assert.Equal(t, true, vars.Get("flag"))
	})

	t.Run("Set overwrites pending", func(t *testing.T) {
		vars := runner.NewContextVariablesWithResolver(
			map[string]any{"key": "pending"},
			func(s string) (string, error) { return "resolved", nil },
		)
		vars.Set("key", "direct")
		assert.Equal(t, "direct", vars.Get("key"))
	})
}

// TestContextVariables_Clone tests cloning behavior.
func TestContextVariables_Clone(t *testing.T) {
	t.Run("clone is independent", func(t *testing.T) {
		original := runner.NewContextVariables(map[string]any{"a": 1})
		clone := original.Clone()

		clone.Set("b", 2)
		original.Set("c", 3)

		assert.Nil(t, original.Get("b"), "original should not have clone's value")
		assert.Nil(t, clone.Get("c"), "clone should not have original's new value")
		assert.Equal(t, 1, clone.Get("a"), "clone should have original's initial value")
	})

	t.Run("clone preserves resolver", func(t *testing.T) {
		resolver := func(s string) (string, error) { return "R:" + s, nil }
		original := runner.NewContextVariablesWithResolver(
			map[string]any{"key": "val"},
			resolver,
		)
		clone := original.Clone()

		assert.Equal(t, "R:val", clone.Get("key"))
	})
}

// TestContextVariables_Walk tests the Walk method.
func TestContextVariables_Walk(t *testing.T) {
	t.Run("walks evaluated values only", func(t *testing.T) {
		vars := runner.NewContextVariablesWithResolver(
			map[string]any{"pending": "value"},
			func(s string) (string, error) { return "resolved", nil },
		)
		vars.Set("evaluated", "direct")

		collected := make(map[string]any)
		vars.Walk(func(k string, v any) {
			collected[k] = v
		})
		assert.Equal(t, map[string]any{"evaluated": "direct"}, collected)
	})

	t.Run("walks all values after resolution", func(t *testing.T) {
		vars := runner.NewContextVariables(map[string]any{"key": "value"})

		collected := make(map[string]any)
		vars.Walk(func(k string, v any) {
			collected[k] = v
		})
		assert.Equal(t, "value", collected["key"])
	})
}

// TestContextVariables_CycleDetection tests that circular dependencies fail deterministically.
func TestContextVariables_CycleDetection(t *testing.T) {
	t.Run("direct self-reference", func(t *testing.T) {
		// a depends on itself: a resolves → calls Get("a") → cycle
		vars := runner.NewContextVariablesWithResolver(
			map[string]any{"a": "${{a}}"},
			nil, // set below
		)
		// We need the resolver to call back into vars.Get.
		// Use SetResolver to wire it up after construction.
		vars.SetResolver(func(s string) (string, error) {
			// Simulate interpolation: "${{a}}" → Get("a")
			if s == "${{a}}" {
				val := vars.Get("a")
				if val == nil {
					return "", nil
				}
				return val.(string), nil
			}
			return s, nil
		})

		err := vars.ResolveAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular variable dependency")
	})

	t.Run("mutual cycle a->b->a", func(t *testing.T) {
		vars := runner.NewContextVariablesWithResolver(
			map[string]any{
				"a": "${{b}}",
				"b": "${{a}}",
			},
			nil,
		)
		vars.SetResolver(func(s string) (string, error) {
			switch s {
			case "${{b}}":
				val := vars.Get("b")
				if val == nil {
					return "", nil
				}
				return val.(string), nil
			case "${{a}}":
				val := vars.Get("a")
				if val == nil {
					return "", nil
				}
				return val.(string), nil
			}
			return s, nil
		})

		err := vars.ResolveAll()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular variable dependency")
	})

	t.Run("chain without cycle resolves fine", func(t *testing.T) {
		// a depends on b, b depends on c, c is a literal
		vars := runner.NewContextVariablesWithResolver(
			map[string]any{
				"a": "use-b",
				"b": "use-c",
				"c": "final",
			},
			nil,
		)
		vars.SetResolver(func(s string) (string, error) {
			switch s {
			case "use-b":
				return vars.Get("b").(string), nil
			case "use-c":
				return vars.Get("c").(string), nil
			}
			return s, nil
		})

		err := vars.ResolveAll()
		assert.NoError(t, err)
		assert.Equal(t, "final", vars.Get("a"))
		assert.Equal(t, "final", vars.Get("b"))
		assert.Equal(t, "final", vars.Get("c"))
	})
}

// TestContextVariables_LoopPattern tests the iteration/loop variable pattern.
func TestContextVariables_LoopPattern(t *testing.T) {
	t.Run("clone and set for each iteration", func(t *testing.T) {
		// Parent context with items list
		parent := runner.NewContextVariables(map[string]any{
			"items":  []any{"a", "b", "c"},
			"prefix": "item:",
		})

		items := []string{"a", "b", "c"}
		results := make([]string, len(items))

		for idx, item := range items {
			// Clone for this iteration
			iterVars := parent.Clone()
			iterVars.Set("item", item)
			iterVars.Set("index", idx)

			// Simulate command using loop vars
			prefix := iterVars.Get("prefix").(string)
			itemVal := iterVars.Get("item").(string)
			results[idx] = prefix + itemVal
		}

		assert.Equal(t, []string{"item:a", "item:b", "item:c"}, results)

		// Parent should be unchanged
		assert.Nil(t, parent.Get("item"), "parent should not have loop var")
		assert.Nil(t, parent.Get("index"), "parent should not have loop var")
	})

	t.Run("loop var shadows parent var", func(t *testing.T) {
		parent := runner.NewContextVariables(map[string]any{
			"item": "parent-value",
		})

		iterVars := parent.Clone()
		iterVars.Set("item", "loop-value")

		assert.Equal(t, "loop-value", iterVars.Get("item"))
		assert.Equal(t, "parent-value", parent.Get("item"))
	})
}
