package treeview_test

import (
	"testing"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/treeview"
)

// TestExecutionTree checks jobs join the root, a nested one conditional, and
// the rendered tree holds one line per node.
func TestExecutionTree(t *testing.T) {
	t.Parallel()

	var tree *treeview.ExecutionTree = treeview.NewExecutionTree("pipeline")
	assert.Equal(t, treeview.StatusRunning, tree.GetStatus())

	build := tree.AddJob(&model.Job{Name: "build"})
	nested := tree.AddJob(&model.Job{Name: "go:build", Nested: true})
	deploy := tree.AddJobWithDeps("deploy", []string{"build"})

	assert.Equal(t, treeview.StatusPending, build.GetStatus())
	assert.Equal(t, treeview.StatusConditional, nested.GetStatus())
	assert.Equal(t, []string{"build"}, deploy.GetDependencies())

	assert.Equal(t, 4, tree.CountLines())
	assert.Equal(t, 4, treeview.CountLines(tree.Node))

	rendered := colors.StripANSI(tree.RenderTree())
	for _, name := range []string{"pipeline", "build", "go:build", "deploy"} {
		assert.Contains(t, rendered, name)
	}
}
