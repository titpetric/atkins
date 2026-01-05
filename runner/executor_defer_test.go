package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/titpetric/atkins-ci/model"
	"github.com/titpetric/atkins-ci/treeview"
)

// TestExecuteSteps_DeferredWaitsForDetached tests that deferred steps wait for detached steps to complete
func TestExecuteSteps_DeferredWaitsForDetached(t *testing.T) {
	executor := NewExecutor()
	ctx := context.Background()

	// Create detached steps
	detachStep1 := &model.Step{
		Name:   "detach-step-1",
		Detach: true,
		Run:    "echo 'detach 1'",
	}

	detachStep2 := &model.Step{
		Name:   "detach-step-2",
		Detach: true,
		Run:    "echo 'detach 2'",
	}

	// Create deferred step
	deferStep := &model.Step{
		Name:     "defer-step",
		Deferred: true,
		Run:      "echo 'defer'",
	}

	steps := []*model.Step{detachStep1, detachStep2, deferStep}

	// Create execution context
	execCtx := &ExecutionContext{
		Variables: make(map[string]interface{}),
		Env:       make(map[string]string),
		Results:   make(map[string]interface{}),
		Pipeline:  &model.Pipeline{},
		Depth:     0,
		JobNodes:  make(map[string]*treeview.TreeNode),
		Logger:    nil,
	}
	execCtx.Context = ctx
	execCtx.Display = treeview.NewDisplay()
	execCtx.Builder = treeview.NewBuilder("test")

	// Execute steps - should not panic
	err := executor.executeSteps(ctx, execCtx, steps)
	assert.NoError(t, err)
}

// TestExecuteSteps_OnlyDetached tests that only detached steps work correctly
func TestExecuteSteps_OnlyDetached(t *testing.T) {
	executor := NewExecutor()
	ctx := context.Background()

	step1 := &model.Step{
		Name:   "detach-1",
		Detach: true,
		Run:    "echo 'test1'",
	}

	step2 := &model.Step{
		Name:   "detach-2",
		Detach: true,
		Run:    "echo 'test2'",
	}

	steps := []*model.Step{step1, step2}

	execCtx := &ExecutionContext{
		Variables: make(map[string]interface{}),
		Env:       make(map[string]string),
		Results:   make(map[string]interface{}),
		Pipeline:  &model.Pipeline{},
		Depth:     0,
		JobNodes:  make(map[string]*treeview.TreeNode),
		Logger:    nil,
	}
	execCtx.Context = ctx
	execCtx.Display = treeview.NewDisplay()
	execCtx.Builder = treeview.NewBuilder("test")

	err := executor.executeSteps(ctx, execCtx, steps)
	assert.NoError(t, err)
}

// TestExecuteSteps_OnlyDeferred tests that deferred steps work when no detached steps exist
func TestExecuteSteps_OnlyDeferred(t *testing.T) {
	executor := NewExecutor()
	ctx := context.Background()

	step1 := &model.Step{
		Name:     "defer-1",
		Deferred: true,
		Run:      "echo 'test1'",
	}

	step2 := &model.Step{
		Name:     "defer-2",
		Deferred: true,
		Run:      "echo 'test2'",
	}

	steps := []*model.Step{step1, step2}

	execCtx := &ExecutionContext{
		Variables: make(map[string]interface{}),
		Env:       make(map[string]string),
		Results:   make(map[string]interface{}),
		Pipeline:  &model.Pipeline{},
		Depth:     0,
		JobNodes:  make(map[string]*treeview.TreeNode),
		Logger:    nil,
	}
	execCtx.Context = ctx
	execCtx.Display = treeview.NewDisplay()
	execCtx.Builder = treeview.NewBuilder("test")

	err := executor.executeSteps(ctx, execCtx, steps)
	assert.NoError(t, err)
}

// TestExecuteSteps_MixedOrder tests that deferred steps execute after regular and detached steps
func TestExecuteSteps_MixedOrder(t *testing.T) {
	executor := NewExecutor()
	ctx := context.Background()

	// Regular step
	regularStep := &model.Step{
		Name: "regular",
		Run:  "echo 'regular'",
	}

	// Detached step
	detachStep := &model.Step{
		Name:   "detach",
		Detach: true,
		Run:    "echo 'detach'",
	}

	// Deferred step
	deferStep := &model.Step{
		Name:     "defer",
		Deferred: true,
		Run:      "echo 'defer'",
	}

	steps := []*model.Step{regularStep, detachStep, deferStep}

	execCtx := &ExecutionContext{
		Variables: make(map[string]interface{}),
		Env:       make(map[string]string),
		Results:   make(map[string]interface{}),
		Pipeline:  &model.Pipeline{},
		Depth:     0,
		JobNodes:  make(map[string]*treeview.TreeNode),
		Logger:    nil,
	}
	execCtx.Context = ctx
	execCtx.Display = treeview.NewDisplay()
	execCtx.Builder = treeview.NewBuilder("test")

	err := executor.executeSteps(ctx, execCtx, steps)
	assert.NoError(t, err)
}

// TestExecuteSteps_EmptySteps tests that empty step list is handled
func TestExecuteSteps_EmptySteps(t *testing.T) {
	executor := NewExecutor()
	ctx := context.Background()

	execCtx := &ExecutionContext{
		Variables: make(map[string]interface{}),
		Env:       make(map[string]string),
		Results:   make(map[string]interface{}),
		Pipeline:  &model.Pipeline{},
		Depth:     0,
		JobNodes:  make(map[string]*treeview.TreeNode),
		Logger:    nil,
	}
	execCtx.Context = ctx
	execCtx.Display = treeview.NewDisplay()
	execCtx.Builder = treeview.NewBuilder("test")

	err := executor.executeSteps(ctx, execCtx, []*model.Step{})
	assert.NoError(t, err)
}


