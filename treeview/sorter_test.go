package treeview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSortJobsByDepth_SingleLevel tests sorting with only root-level jobs
func TestSortJobsByDepth_SingleLevel(t *testing.T) {
	t.Run("alphabetical order for root jobs", func(t *testing.T) {
		jobs := []string{"zebra", "apple", "banana"}
		result := SortJobsByDepth(jobs)

		expected := []string{"apple", "banana", "zebra"}
		assert.Equal(t, expected, result)
	})

	t.Run("already sorted", func(t *testing.T) {
		jobs := []string{"apple", "banana", "zebra"}
		result := SortJobsByDepth(jobs)

		expected := []string{"apple", "banana", "zebra"}
		assert.Equal(t, expected, result)
	})

	t.Run("single job", func(t *testing.T) {
		jobs := []string{"test"}
		result := SortJobsByDepth(jobs)

		expected := []string{"test"}
		assert.Equal(t, expected, result)
	})

	t.Run("empty list", func(t *testing.T) {
		jobs := []string{}
		result := SortJobsByDepth(jobs)

		expected := []string{}
		assert.Equal(t, expected, result)
	})
}

// TestSortJobsByDepth_MultiLevel tests sorting with nested jobs (depth > 0)
func TestSortJobsByDepth_MultiLevel(t *testing.T) {
	t.Run("simple nested jobs", func(t *testing.T) {
		jobs := []string{"test:run", "test", "docker:run"}
		result := SortJobsByDepth(jobs)

		// Expected: depth 0 first (test), then depth 1 alphabetically (docker:run, test:run)
		expected := []string{"test", "docker:run", "test:run"}
		assert.Equal(t, expected, result)
	})

	t.Run("deeply nested jobs", func(t *testing.T) {
		jobs := []string{
			"test:run:subtask",
			"test",
			"test:run",
			"docker:setup:init",
		}
		result := SortJobsByDepth(jobs)

		// Depth 0: test
		// Depth 1: test:run (alphabetically)
		// Depth 2: docker:setup:init, test:run:subtask (alphabetically)
		expected := []string{
			"test",
			"test:run",
			"docker:setup:init",
			"test:run:subtask",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("all nested at same depth", func(t *testing.T) {
		jobs := []string{"zebra:test", "apple:test", "banana:test"}
		result := SortJobsByDepth(jobs)

		expected := []string{"apple:test", "banana:test", "zebra:test"}
		assert.Equal(t, expected, result)
	})
}

// TestSortJobsByDepth_RealWorld tests sorting with realistic job names
func TestSortJobsByDepth_RealWorld(t *testing.T) {
	t.Run("mixed root and nested jobs", func(t *testing.T) {
		jobs := []string{
			"build",
			"build:run",
			"build:run:compile",
			"docker:clean",
			"docker:setup",
			"test",
			"test:coverage",
			"test:integration",
			"test:run",
		}

		result := SortJobsByDepth(jobs)

		// Expected ordering:
		// Depth 0: build, test (alphabetically)
		// Depth 1: build:run, docker:clean, docker:setup, test:coverage, test:integration, test:run
		// Depth 2: build:run:compile
		expected := []string{
			"build",
			"test",
			"build:run",
			"docker:clean",
			"docker:setup",
			"test:coverage",
			"test:integration",
			"test:run",
			"build:run:compile",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("with multiple root jobs", func(t *testing.T) {
		jobs := []string{
			"lint",
			"format:check",
			"build",
			"test",
		}
		result := SortJobsByDepth(jobs)

		expected := []string{"build", "lint", "test", "format:check"}
		assert.Equal(t, expected, result)
	})

	t.Run("complex nested structure", func(t *testing.T) {
		jobs := []string{
			"deploy:prod:finalize",
			"deploy",
			"deploy:prod",
			"test:unit",
			"test",
			"build",
		}
		result := SortJobsByDepth(jobs)

		expected := []string{
			"build",
			"deploy",
			"test",
			"deploy:prod",
			"test:unit",
			"deploy:prod:finalize",
		}
		assert.Equal(t, expected, result)
	})
}

// TestSortJobsByDepth_Consistency tests that sorting is stable and consistent
func TestSortJobsByDepth_Consistency(t *testing.T) {
	t.Run("multiple sorts produce same result", func(t *testing.T) {
		original := []string{"test:run", "test", "docker:run", "build", "build:run"}

		result1 := SortJobsByDepth(original)
		result2 := SortJobsByDepth(result1)

		assert.Equal(t, result1, result2, "second sort should produce same result")
	})

	t.Run("different input order same result", func(t *testing.T) {
		input1 := []string{"test", "docker:run", "test:run", "build"}
		input2 := []string{"docker:run", "test:run", "build", "test"}

		result1 := SortJobsByDepth(input1)
		result2 := SortJobsByDepth(input2)

		assert.Equal(t, result1, result2, "different input orders should produce same result")
	})
}

// TestSortJobsByDepth_DoesNotMutate tests that SortJobsByDepth doesn't mutate input
func TestSortJobsByDepth_DoesNotMutate(t *testing.T) {
	t.Run("input slice not modified", func(t *testing.T) {
		original := []string{"test:run", "test", "docker:run"}
		originalCopy := make([]string, len(original))
		copy(originalCopy, original)

		SortJobsByDepth(original)

		assert.Equal(t, originalCopy, original, "input should not be modified")
	})
}

// TestCountDepth tests the depth counting logic
func TestCountDepth(t *testing.T) {
	tests := []struct {
		name          string
		jobName       string
		expectedDepth int
	}{
		{"root level job", "test", 0},
		{"single nested", "test:run", 1},
		{"double nested", "test:run:subtask", 2},
		{"triple nested", "a:b:c:d", 3},
		{"complex name", "docker:compose:up", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countDepth(tt.jobName)
			assert.Equal(t, tt.expectedDepth, result)
		})
	}
}

// TestCompareByDepthThenName tests the comparison function directly
func TestCompareByDepthThenName(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int // -1: a < b, 0: a == b, 1: a > b
	}{
		// Same depth (0), alphabetical comparison
		{"root alphabetic a<b", "apple", "banana", -1},
		{"root alphabetic b<a", "zebra", "apple", 1},
		{"same root job", "test", "test", 0},

		// Different depth
		{"depth 0 vs 1", "test", "test:run", -1},
		{"depth 1 vs 0", "test:run", "test", 1},
		{"depth 1 vs 2", "test:run", "test:run:subtask", -1},

		// Same depth, different names
		{"depth 1 alphabetic", "docker:setup", "test:run", -1},
		{"depth 1 alphabetic reverse", "test:run", "docker:setup", 1},

		// Same depth, same base but different sub
		{"same parent different sub", "test:integration", "test:unit", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareByDepthThenName(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "comparing %q vs %q", tt.a, tt.b)
		})
	}
}
