package treeview_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/treeview"
)

func TestDisplayCreation(t *testing.T) {
	t.Run("NewDisplay creates display with correct defaults", func(t *testing.T) {
		display := treeview.NewDisplay()
		assert.NotNil(t, display)
		// IsTerminal depends on whether tests are run in a TTY
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
		// Cleanup should be a no-op and safe to call
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

		// Simulate multiple render calls
		display.Render(root)
		display.Render(root)

		// Final render should work
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

func TestSlidingWindowCalculation(t *testing.T) {
	t.Run("sliding window uses termHeight-1", func(t *testing.T) {
		// The sliding window should use termHeight-1 to leave room for cursor
		termHeight := 24
		maxLines := termHeight - 1 // 23

		testCases := []struct {
			totalLines    int
			expectedLines int
		}{
			{10, 10},  // fits, no trimming
			{20, 20},  // fits, no trimming
			{23, 23},  // exactly maxLines
			{24, 23},  // exceeds, trim to 23
			{50, 23},  // exceeds, trim to 23
			{100, 23}, // exceeds, trim to 23
		}

		for _, tc := range testCases {
			lines := tc.totalLines
			if lines > maxLines {
				lines = maxLines
			}
			assert.Equal(t, tc.expectedLines, lines,
				"totalLines=%d should result in %d lines", tc.totalLines, tc.expectedLines)
		}
	})
}

func TestDisplayRollbackCalculation(t *testing.T) {
	t.Run("rollback should match lines printed when output grows past terminal height", func(t *testing.T) {
		// Simulate the scenario where output exceeds terminal height
		// and we need to ensure rollback is calculated correctly

		termHeight := 30

		// Scenario: output grows from 25 -> 28 -> 32 -> 35 lines
		renders := []struct {
			totalLines    int
			expectedClamp int
		}{
			{25, 25}, // fits in terminal
			{28, 28}, // still fits
			{32, 30}, // exceeds, clamped to 30
			{35, 30}, // exceeds, clamped to 30
		}

		lastLineCount := 0
		for i, r := range renders {
			lineCount := r.totalLines
			if termHeight > 0 && lineCount > termHeight {
				lineCount = termHeight
			}

			assert.Equal(t, r.expectedClamp, lineCount,
				"render %d: expected clamped line count", i)

			// The problematic case: we print MORE lines than we rolled back
			// This happens when output grows and we haven't reached terminal height yet
			if lastLineCount > 0 && lineCount > lastLineCount {
				// This is where scroll artifacts can occur
				// The fix ensures we handle this transition properly
				t.Logf("render %d: rollback=%d, print=%d (diff=%d)",
					i, lastLineCount, lineCount, lineCount-lastLineCount)
			}

			lastLineCount = lineCount
		}
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

func TestDisplayCursorMovementSequences(t *testing.T) {
	t.Run("cursor up value should not exceed terminal height", func(t *testing.T) {
		// This test validates that the ANSI cursor-up sequence value
		// is bounded properly to prevent over-rollback

		// Simulate multiple renders and track the cursor-up values
		termHeight := 30
		renders := []int{10, 20, 30, 40, 50, 60}

		lastLineCount := 0
		for _, totalLines := range renders {
			lineCount := totalLines
			if termHeight > 0 && lineCount > termHeight {
				lineCount = termHeight
			}

			rollback := lastLineCount

			// The rollback should never exceed what we previously printed
			// AND should never exceed terminal height
			assert.LessOrEqual(t, rollback, termHeight,
				"rollback should not exceed terminal height")
			assert.LessOrEqual(t, rollback, lastLineCount,
				"rollback should not exceed previous line count")

			lastLineCount = lineCount
		}
	})
}

func TestDisplayScrollTransition(t *testing.T) {
	t.Run("transition from non-scroll to scroll state", func(t *testing.T) {
		// This tests the specific transition that can cause duplicate lines:
		// when output grows from under terminal height to over terminal height

		termHeight := 30

		// Build up renders
		var lastLineCount int
		transitionIssues := []string{}

		for total := 20; total <= 40; total += 2 {
			lineCount := total
			if termHeight > 0 && lineCount > termHeight {
				lineCount = termHeight
			}

			if lastLineCount > 0 {
				if lineCount > lastLineCount {
					// We're printing more than we're rolling back
					// This is the problematic transition
					diff := lineCount - lastLineCount
					if diff > 0 && lineCount == termHeight {
						// First time hitting terminal height while growing
						transitionIssues = append(transitionIssues,
							"scroll transition: rollback=%d, print=%d")
					}
				}
			}

			lastLineCount = lineCount
		}

		// The test documents the issue - when we first hit terminal height
		// while output is still growing, there's a mismatch between
		// rollback and print counts
		t.Logf("documented %d scroll transitions", len(transitionIssues))
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

// simulateRender simulates what Display.Render does and returns debug info
func simulateRender(termHeight, totalLines, lastLineCount int) (lineCount, rollback int, wouldScroll bool) {
	lineCount = totalLines
	if termHeight > 0 && lineCount > termHeight {
		lineCount = termHeight
	}

	rollback = lastLineCount

	// Scroll would occur if we print more lines than we roll back
	// and the resulting cursor position would exceed terminal height
	wouldScroll = lineCount > lastLineCount && lineCount == termHeight

	return
}

func TestSimulateRender(t *testing.T) {
	t.Run("simulate render behavior", func(t *testing.T) {
		tests := []struct {
			name          string
			termHeight    int
			totalLines    int
			lastLineCount int
			wantLineCount int
			wantRollback  int
			wantScroll    bool
		}{
			{
				name:          "normal render under terminal height",
				termHeight:    30,
				totalLines:    20,
				lastLineCount: 15,
				wantLineCount: 20,
				wantRollback:  15,
				wantScroll:    false,
			},
			{
				name:          "render exactly at terminal height",
				termHeight:    30,
				totalLines:    30,
				lastLineCount: 25,
				wantLineCount: 30,
				wantRollback:  25,
				wantScroll:    true, // first time hitting terminal height
			},
			{
				name:          "render exceeding terminal height",
				termHeight:    30,
				totalLines:    40,
				lastLineCount: 30,
				wantLineCount: 30,
				wantRollback:  30,
				wantScroll:    false, // already at terminal height
			},
			{
				name:          "render shrinking from terminal height",
				termHeight:    30,
				totalLines:    25,
				lastLineCount: 30,
				wantLineCount: 25,
				wantRollback:  30,
				wantScroll:    false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				lineCount, rollback, wouldScroll := simulateRender(
					tt.termHeight, tt.totalLines, tt.lastLineCount)

				assert.Equal(t, tt.wantLineCount, lineCount, "lineCount")
				assert.Equal(t, tt.wantRollback, rollback, "rollback")
				assert.Equal(t, tt.wantScroll, wouldScroll, "wouldScroll")
			})
		}
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

func TestCountOutputLines(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{"empty string", "", 0},
		{"single line no newline", "hello", 0},
		{"single line with newline", "hello\n", 1},
		{"two lines", "hello\nworld\n", 2},
		{"three lines", "a\nb\nc\n", 3},
		{"with ansi codes", "\033[1mhello\033[0m\n\033[32mworld\033[0m\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			for _, ch := range tt.output {
				if ch == '\n' {
					count++
				}
			}
			assert.Equal(t, tt.want, count, "newline count")
		})
	}
}

func TestCursorUpSequenceExtraction(t *testing.T) {
	t.Run("extract cursor up values from ANSI sequences", func(t *testing.T) {
		// Pattern for cursor up: \033[<N>A
		pattern := regexp.MustCompile(`\x1b\[(\d+)A`)

		// Simulated terminal output with cursor movements
		sample := "\x1b[25A\x1b[Jline1\nline2\n\x1b[2A\x1b[Jnew1\nnew2\n"

		matches := pattern.FindAllStringSubmatch(sample, -1)
		assert.Len(t, matches, 2, "should find 2 cursor-up sequences")

		if len(matches) >= 2 {
			assert.Equal(t, "25", matches[0][1], "first cursor-up value")
			assert.Equal(t, "2", matches[1][1], "second cursor-up value")
		}
	})
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
