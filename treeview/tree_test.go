package treeview_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/treeview"
)

// TestTreeNode checks steps join a job as running children, a deferred one
// marked, and SetStatus reaches the node.
func TestTreeNode(t *testing.T) {
	t.Parallel()

	job := treeview.NewTreeNode("build")
	step := job.AddStep("run: go build")
	deferred := job.AddStepDeferred("run: cleanup")

	children := job.GetChildren()
	assert.Len(t, children, 2)
	assert.Equal(t, "run: go build", children[0].GetName())
	assert.Equal(t, treeview.StatusRunning, step.GetStatus())
	assert.True(t, deferred.Deferred)

	deferred.SetStatus(treeview.StatusPassed)
	assert.Equal(t, treeview.StatusPassed, deferred.GetStatus())
	assert.False(t, deferred.Deferred)
}
