package treeview

import (
	"github.com/titpetric/atkins/model"
)

// ExecutionTree holds the entire execution tree.
type ExecutionTree struct {
	*TreeNode
}

// NewExecutionTree creates a new execution tree with a root node.
func NewExecutionTree(pipelineName string) *ExecutionTree {
	return &ExecutionTree{
		TreeNode: &TreeNode{
			Node: &Node{
				Name:     pipelineName,
				Status:   StatusRunning,
				Children: make([]*Node, 0),
			},
		},
	}
}

// AddJob adds a job node to the tree.
func (et *ExecutionTree) AddJob(job *model.Job) *TreeNode {
	et.Lock()
	defer et.Unlock()

	status := StatusPending
	if job.Nested {
		status = StatusConditional
	}

	node := &TreeNode{
		Node: &Node{
			Name:         job.Name,
			Status:       status,
			Children:     make([]*Node, 0),
			Dependencies: make([]string, 0),
		},
	}
	et.Children = append(et.Children, node.Node)
	return node
}

// AddJobWithDeps adds a job node to the tree with dependencies.
func (et *ExecutionTree) AddJobWithDeps(jobName string, deps []string) *TreeNode {
	et.Lock()
	defer et.Unlock()

	node := &TreeNode{
		Node: &Node{
			Name:         jobName,
			Status:       StatusPending,
			Children:     make([]*Node, 0),
			Dependencies: deps,
		},
	}
	et.Children = append(et.Children, node.Node)
	return node
}

// RenderTree renders the entire tree to a string (live rendering).
func (et *ExecutionTree) RenderTree() string {
	et.Lock()
	defer et.Unlock()

	renderer := NewRenderer()
	return renderer.Render(et.Node)
}

// CountLines returns the number of lines the tree will render.
func (et *ExecutionTree) CountLines() int {
	et.Lock()
	defer et.Unlock()

	return CountLines(et.Node)
}
