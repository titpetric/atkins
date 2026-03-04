package treeview

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/colors"
)

func TestRenderNodeSummary(t *testing.T) {
	t.Run("summarized node shows progress counter", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("for-loop-step")
		step.Summarize = true

		child1 := NewNode("iter-1")
		child1.SetStatus(StatusPassed)
		step.AddChild(child1)

		child2 := NewNode("iter-2")
		child2.SetStatus(StatusPassed)
		step.AddChild(child2)

		child3 := NewNode("iter-3")
		child3.SetStatus(StatusRunning)
		step.AddChild(child3)

		root.AddChild(step)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		assert.Contains(t, stripped, "2/3", "summary should show passing/total progress counter")
	})

	t.Run("summary includes failed in total", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("for-loop-step")
		step.Summarize = true

		child1 := NewNode("iter-1")
		child1.SetStatus(StatusPassed)
		step.AddChild(child1)

		child2 := NewNode("iter-2")
		child2.SetStatus(StatusFailed)
		step.AddChild(child2)

		child3 := NewNode("iter-3")
		child3.SetStatus(StatusPassed)
		step.AddChild(child3)

		root.AddChild(step)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		assert.Contains(t, stripped, "2/3", "total should include failed items")
		assert.NotContains(t, stripped, "2/2", "total should not exclude failed items")
	})

	t.Run("summary with no children shows no counter", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("empty-step")
		step.Summarize = true

		root.AddChild(step)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		assert.NotContains(t, stripped, "/", "no progress counter when no children")
		assert.Contains(t, stripped, "empty-step", "should still show node name")
	})

	t.Run("static render also shows summary", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("for-loop-step")
		step.Summarize = true

		child1 := NewNode("iter-1")
		child1.SetStatus(StatusPassed)
		step.AddChild(child1)

		child2 := NewNode("iter-2")
		child2.SetStatus(StatusRunning)
		step.AddChild(child2)

		root.AddChild(step)

		output := renderer.RenderStatic(root)
		stripped := colors.StripANSI(output)

		assert.Contains(t, stripped, "1/2", "static render should also show progress counter")
	})
}

func TestRenderOutputLineCount(t *testing.T) {
	t.Run("single output line adds 1 line", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("run: echo hello")
		step.SetStatus(StatusPassed)
		step.SetOutput([]string{"hello"})
		root.AddChild(step)

		output := renderer.Render(root)
		lines := countOutputLines(output)

		// root(1) + step(1) + output(1) = 3
		assert.Equal(t, 3, lines, "single output line should add 1 line to render")
	})

	t.Run("multiple output lines add N+2 lines for borders", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("run: coverfunc")
		step.SetStatus(StatusPassed)
		step.SetOutput([]string{"line1", "line2", "line3"})
		root.AddChild(step)

		output := renderer.Render(root)
		lines := countOutputLines(output)

		// root(1) + step(1) + top_border(1) + 3_content + bottom_border(1) = 7
		assert.Equal(t, 7, lines, "3 output lines should add 5 lines (3 content + 2 borders)")
	})

	t.Run("output lines counted correctly with children", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		job := NewNode("test:mergecov")
		job.SetStatus(StatusPassed)

		step1 := NewNode("run: merge")
		step1.SetStatus(StatusPassed)
		step1.SetOutput([]string{"Merged 288 files"})
		job.AddChild(step1)

		step2 := NewNode("run: coverfunc --packages")
		step2.SetStatus(StatusPassed)
		step2.SetOutput([]string{"pkg1, 90%", "pkg2, 80%", "pkg3, 70%"})
		job.AddChild(step2)

		step3 := NewNode("run: cover -html")
		step3.SetStatus(StatusPassed)
		job.AddChild(step3)

		root.AddChild(job)

		output := renderer.Render(root)
		lines := countOutputLines(output)

		// root(1) + job(1) + step1(1) + step1_output(1) + step2(1) + step2_borders(2) + step2_content(3) + step3(1) = 11
		assert.Equal(t, 11, lines, "line count must include output content and borders")
	})

	t.Run("output added between renders changes line count", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("run: coverfunc")
		step.SetStatus(StatusRunning)
		root.AddChild(step)

		// Render without output
		output1 := renderer.Render(root)
		lines1 := countOutputLines(output1)

		// root(1) + step(1) = 2
		assert.Equal(t, 2, lines1, "render without output")

		// Add output (simulating passthru capture completing)
		step.SetStatus(StatusPassed)
		step.SetOutput([]string{"pkg1, 90%", "pkg2, 80%", "pkg3, 70%", "pkg4, 60%"})

		output2 := renderer.Render(root)
		lines2 := countOutputLines(output2)

		// root(1) + step(1) + top_border(1) + 4_content + bottom_border(1) = 8
		assert.Equal(t, 8, lines2, "render with output must account for content + borders")

		// The difference is exactly the output lines + borders
		assert.Equal(t, 6, lines2-lines1, "output should add exactly N+2 lines for bordered output")
	})

	t.Run("multiple steps with output accumulate correctly", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		job := NewNode("job")
		job.SetStatus(StatusRunning)

		// Step 1 done with output
		step1 := NewNode("step1")
		step1.SetStatus(StatusPassed)
		step1.SetOutput([]string{"output1"})
		job.AddChild(step1)

		// Step 2 done with multi-line output
		step2 := NewNode("step2")
		step2.SetStatus(StatusPassed)
		step2.SetOutput([]string{"a", "b"})
		job.AddChild(step2)

		// Step 3 still running
		step3 := NewNode("step3")
		step3.SetStatus(StatusRunning)
		job.AddChild(step3)

		root.AddChild(job)

		output := renderer.Render(root)
		lines := countOutputLines(output)

		// root(1) + job(1) + step1(1) + step1_output(1) + step2(1) + step2_borders(2) + step2_content(2) + step3(1) = 10
		assert.Equal(t, 10, lines, "accumulated output from multiple steps")
	})
}

func TestConcurrentOutputAndRender(t *testing.T) {
	t.Run("concurrent SetOutput and Render operations", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		// Create a job with multiple steps
		job := NewNode("test:job")
		job.SetStatus(StatusRunning)
		root.AddChild(job)

		// Create steps
		var steps []*Node
		for i := 0; i < 5; i++ {
			step := NewNode("run: step" + string(rune(48+i)))
			step.SetStatus(StatusPending)
			job.AddChild(step)
			steps = append(steps, step)
		}

		// Run concurrent operations
		done := make(chan bool, 20)

		// Multiple goroutines setting output
		for i := 0; i < 5; i++ {
			stepIdx := i
			go func() {
				for j := 0; j < 10; j++ {
					steps[stepIdx].SetOutput([]string{"line1", "line2", "line3"})
					steps[stepIdx].SetStatus(StatusPassed)
				}
				done <- true
			}()
		}

		// Multiple goroutines rendering
		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < 10; j++ {
					output := renderer.Render(root)
					lines := countOutputLines(output)
					// Just verify it doesn't panic and produces valid output
					assert.True(t, lines >= 2, "should have at least root and job lines")
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Final render should be consistent
		output := renderer.Render(root)
		lines := countOutputLines(output)
		assert.True(t, lines > 0, "final render should have lines")
	})

	t.Run("consistent line count after concurrent modifications", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("run: test")
		step.SetStatus(StatusPassed)
		root.AddChild(step)

		// Set output with known content
		step.SetOutput([]string{"line1", "line2", "line3"})

		// Render multiple times and verify consistent line count
		expectedLines := 0
		for i := 0; i < 10; i++ {
			output := renderer.Render(root)
			lines := countOutputLines(output)
			if expectedLines == 0 {
				expectedLines = lines
			}
			assert.Equal(t, expectedLines, lines, "line count should be consistent across renders")
		}

		// Should be: root(1) + step(1) + border(1) + 3 content + border(1) = 7
		assert.Equal(t, 7, expectedLines, "expected 7 lines for 3-line output with borders")
	})
}

func TestRenderProgressCounter(t *testing.T) {
	t.Run("expanded node with multiple children shows progress", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("task: test:detail")

		child1 := NewNode("test:detail (item: a)")
		child1.SetStatus(StatusPassed)
		step.AddChild(child1)

		child2 := NewNode("test:detail (item: b)")
		child2.SetStatus(StatusPassed)
		step.AddChild(child2)

		child3 := NewNode("test:detail (item: c)")
		child3.SetStatus(StatusRunning)
		step.AddChild(child3)

		root.AddChild(step)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		assert.Contains(t, stripped, "2/3", "expanded node should show progress counter")
		assert.Contains(t, stripped, "test:detail (item: a)", "children should still be visible")
		assert.Contains(t, stripped, "test:detail (item: b)", "children should still be visible")
		assert.Contains(t, stripped, "test:detail (item: c)", "children should still be visible")
	})

	t.Run("expanded node all passed shows full count", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("task: test:detail")
		for i := 0; i < 5; i++ {
			child := NewNode("iter")
			child.SetStatus(StatusPassed)
			step.AddChild(child)
		}

		root.AddChild(step)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		assert.Contains(t, stripped, "5/5", "all passed should show full count")
	})

	t.Run("single child does not show counter", func(t *testing.T) {
		renderer := NewRenderer()
		root := NewNode("pipeline")

		step := NewNode("task: test:simple")
		child := NewNode("run: echo hello")
		child.SetStatus(StatusPassed)
		step.AddChild(child)

		root.AddChild(step)

		output := renderer.Render(root)
		stripped := colors.StripANSI(output)

		assert.NotContains(t, stripped, "1/1", "single child should not show counter")
	})
}
