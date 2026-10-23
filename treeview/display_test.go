package treeview_test

import (
	"strings"
	"testing"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/treeview"
)

func TestDisplayCreation(t *testing.T) {
	t.Run("NewDisplay creates display with correct defaults", func(t *testing.T) {
		display := treeview.NewDisplay()
		assert.NotNil(t, display)
		// IsTerminal depends on whether the tests run in a TTY
	})

	t.Run("NewDisplayWithFinal creates display with finalOnly mode", func(t *testing.T) {
		display := treeview.NewDisplayWithFinal(true)
		assert.NotNil(t, display)
		// In finalOnly mode, IsTerminal returns false
		assert.False(t, display.IsTerminal())
	})

	t.Run("NewDisplayWithFinal false preserves terminal detection", func(t *testing.T) {
		display := treeview.NewDisplayWithFinal(false)
		assert.NotNil(t, display)
		// IsTerminal depends on actual terminal
	})

	t.Run("NewSilentDisplay creates non-terminal display", func(t *testing.T) {
		display := treeview.NewSilentDisplay()
		assert.NotNil(t, display)
		assert.False(t, display.IsTerminal())
	})
}

func TestDisplayInvalidate(t *testing.T) {
	t.Run("Invalidate resets line counter", func(t *testing.T) {
		display := treeview.NewSilentDisplay()
		// Invalidate should not panic on fresh display
		display.Invalidate()
	})
}

func TestDisplayCleanup(t *testing.T) {
	t.Run("Cleanup is safe to call multiple times", func(t *testing.T) {
		display := treeview.NewSilentDisplay()
		// Cleanup is a no-op and safe to call
		display.Cleanup()
		display.Cleanup()
	})

	t.Run("Cleanup is safe on fresh display", func(t *testing.T) {
		display := treeview.NewDisplay()
		display.Cleanup()
	})
}

func TestDisplayRenderFinal(t *testing.T) {
	t.Run("RenderFinal outputs full tree on silent display", func(t *testing.T) {
		display := treeview.NewSilentDisplay()
		root := treeview.NewNode("pipeline")
		step := treeview.NewNode("test:step")
		step.SetStatus(treeview.StatusPassed)
		root.AddChild(step)

		// Should not panic
		display.RenderFinal(root)
	})

	t.Run("RenderFinal works after Render calls", func(t *testing.T) {
		display := treeview.NewSilentDisplay()
		root := treeview.NewNode("pipeline")

		// Simulate repeated render calls
		display.Render(root)
		display.Render(root)

		// Final render succeeds
		display.RenderFinal(root)
	})
}

func TestDisplayRenderStatic(t *testing.T) {
	t.Run("RenderStatic outputs tree", func(t *testing.T) {
		display := treeview.NewSilentDisplay()
		root := treeview.NewNode("pipeline")
		step := treeview.NewNode("test:step")
		step.SetStatus(treeview.StatusPassed)
		root.AddChild(step)

		// Should not panic
		display.RenderStatic(root)
	})
}

func TestDisplayNoDuplicateLines(t *testing.T) {
	t.Run("rendered output should not contain duplicate lines", func(t *testing.T) {
		renderer := treeview.NewRenderer()
		root := treeview.NewNode("pipeline")

		// Create a tree with distinct lines that would exceed a small terminal
		job := treeview.NewNode("test:job")
		job.SetStatus(treeview.StatusPassed)

		for i := 0; i < 20; i++ {
			step := treeview.NewNode("run: unique-step-" + string(rune('A'+i)))
			step.SetStatus(treeview.StatusPassed)
			job.AddChild(step)
		}

		root.AddChild(job)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		// Check for duplicate lines (excluding common tree prefixes)
		lines := strings.Split(stripped, "\n")
		seen := make(map[string]int)

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			// Skip lines that are just tree structure
			if trimmed == "│" || trimmed == "├─" || trimmed == "└─" {
				continue
			}

			if prevIdx, exists := seen[trimmed]; exists {
				t.Errorf("duplicate line found at index %d and %d: %q", prevIdx, i, trimmed)
			}
			seen[trimmed] = i
		}
	})
}

func TestDisplayLineCountAccuracy(t *testing.T) {
	t.Run("line count matches actual newlines in output", func(t *testing.T) {
		renderer := treeview.NewRenderer()
		root := treeview.NewNode("pipeline")

		step := treeview.NewNode("run: test")
		step.SetStatus(treeview.StatusPassed)
		step.SetOutput([]string{"line1", "line2", "line3"})
		root.AddChild(step)

		output := renderer.Render(root)

		// Count newlines
		newlineCount := strings.Count(output, "\n")

		// Count lines after split (remove trailing empty)
		lines := strings.Split(output, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		assert.Equal(t, newlineCount, len(lines),
			"newline count should equal split line count")
	})

	t.Run("output with borders counted correctly", func(t *testing.T) {
		renderer := treeview.NewRenderer()
		root := treeview.NewNode("pipeline")

		step := treeview.NewNode("run: coverfunc")
		step.SetStatus(treeview.StatusPassed)
		step.SetOutput([]string{"pkg1, 90%", "pkg2, 80%", "pkg3, 70%"})
		root.AddChild(step)

		output := renderer.Render(root)
		lines := strings.Split(output, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		// Expected: pipeline(1) + step(1) + top_border(1) + 3_content + bottom_border(1) = 7
		assert.Equal(t, 7, len(lines),
			"3 output lines should produce 7 total lines (node + borders + content)")
	})
}

func TestRenderANSISequences(t *testing.T) {
	t.Run("render output uses correct ANSI structure", func(t *testing.T) {
		renderer := treeview.NewRenderer()
		root := treeview.NewNode("pipeline")

		step := treeview.NewNode("test:step")
		step.SetStatus(treeview.StatusPassed)
		root.AddChild(step)

		output := renderer.Render(root)

		// Output should end with newline
		assert.True(t, strings.HasSuffix(output, "\n"),
			"output should end with newline")

		// Output should not have consecutive newlines (empty lines)
		assert.NotContains(t, output, "\n\n",
			"output should not have empty lines")
	})
}

func TestDisplayOutputCapture(t *testing.T) {
	t.Run("capture and verify no duplicate content in final output", func(t *testing.T) {
		// This test builds a tree progressively and verifies
		// the final rendered output has no duplicate meaningful lines

		renderer := treeview.NewRenderer()

		// Create a tree that would span 40+ lines
		root := treeview.NewNode("pipeline")

		job := treeview.NewNode("test:large-job")
		job.SetStatus(treeview.StatusPassed)

		// Add steps with unique identifiable content
		for i := 0; i < 15; i++ {
			step := treeview.NewNode("run: command-" + string(rune('0'+i/10)) + string(rune('0'+i%10)))
			step.SetStatus(treeview.StatusPassed)
			if i%3 == 0 {
				step.SetOutput([]string{
					"output-" + string(rune('A'+i)) + "-line-1",
					"output-" + string(rune('A'+i)) + "-line-2",
				})
			}
			job.AddChild(step)
		}

		root.AddChild(job)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		// Extract meaningful content lines (not tree structure or borders)
		lines := strings.Split(stripped, "\n")
		var contentLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip empty lines
			if trimmed == "" {
				continue
			}
			// Skip pure tree-structure lines
			if trimmed == "│" {
				continue
			}
			// Skip border lines (they legitimately repeat for each output box)
			if isBorderLine(trimmed) {
				continue
			}
			contentLines = append(contentLines, trimmed)
		}

		// Check for exact duplicates among meaningful content
		seen := make(map[string]bool)
		duplicates := []string{}
		for _, line := range contentLines {
			if seen[line] {
				duplicates = append(duplicates, line)
			}
			seen[line] = true
		}

		assert.Empty(t, duplicates,
			"should have no duplicate meaningful lines in rendered output")
	})
}

// isBorderLine checks if a line is a box border character sequence
func isBorderLine(line string) bool {
	// Check if line contains box drawing characters (these repeat for each output box)
	if strings.Contains(line, "┌") || strings.Contains(line, "┐") ||
		strings.Contains(line, "└") || strings.Contains(line, "┘") ||
		strings.Contains(line, "─") {
		// Line has box drawing characters - check if it's a pure border
		// (not content inside the box)
		inner := strings.Trim(line, "│├└─ ┌┐┘")
		// If after removing borders and tree chars, only whitespace/dashes remain, it's a border
		if strings.Trim(inner, "─ ") == "" {
			return true
		}
	}
	return false
}

// Benchmark to measure rendering performance
func BenchmarkRenderLargeTree(b *testing.B) {
	renderer := treeview.NewRenderer()

	root := treeview.NewNode("pipeline")
	for i := 0; i < 10; i++ {
		job := treeview.NewNode("job-" + string(rune('A'+i)))
		job.SetStatus(treeview.StatusPassed)
		for j := 0; j < 10; j++ {
			step := treeview.NewNode("step-" + string(rune('0'+j)))
			step.SetStatus(treeview.StatusPassed)
			job.AddChild(step)
		}
		root.AddChild(job)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Render(root)
	}
}
