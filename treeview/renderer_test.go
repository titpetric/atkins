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
