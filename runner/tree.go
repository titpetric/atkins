package runner

import (
	"fmt"
	"sync"

	"github.com/titpetric/atkins-ci/colors"
)

// NodeStatus represents the execution status of a node
type NodeStatus int

const (
	StatusPending NodeStatus = iota
	StatusRunning
	StatusPassed
	StatusFailed
)

// TreeNode represents a node in the execution tree
type TreeNode struct {
	Name     string
	Status   NodeStatus
	Spinner  string
	Children []*TreeNode
	mu       sync.Mutex
}

// ExecutionTree holds the entire execution tree
type ExecutionTree struct {
	Root *TreeNode
	mu   sync.Mutex
}

// NewExecutionTree creates a new execution tree with a root node
func NewExecutionTree(pipelineName string) *ExecutionTree {
	return &ExecutionTree{
		Root: &TreeNode{
			Name:     pipelineName,
			Status:   StatusRunning,
			Children: make([]*TreeNode, 0),
		},
	}
}

// AddJob adds a job node to the tree
func (et *ExecutionTree) AddJob(jobName string) *TreeNode {
	et.mu.Lock()
	defer et.mu.Unlock()

	node := &TreeNode{
		Name:     jobName,
		Status:   StatusPending,
		Children: make([]*TreeNode, 0),
	}
	et.Root.Children = append(et.Root.Children, node)
	return node
}

// AddStep adds a step node to a job
func (job *TreeNode) AddStep(stepName string) *TreeNode {
	job.mu.Lock()
	defer job.mu.Unlock()

	node := &TreeNode{
		Name:   stepName,
		Status: StatusRunning,
	}
	job.Children = append(job.Children, node)
	return node
}

// SetStatus updates a node's status
func (node *TreeNode) SetStatus(status NodeStatus) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.Status = status
}

// SetSpinner updates the spinner display
func (node *TreeNode) SetSpinner(spinner string) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.Spinner = spinner
}

// RenderTree renders the entire tree to a string (live rendering)
func (et *ExecutionTree) RenderTree() string {
	et.mu.Lock()
	defer et.mu.Unlock()

	output := colors.BrightGreen(et.Root.Name) + "\n"
	for i, job := range et.Root.Children {
		isLast := i == len(et.Root.Children)-1
		output += renderNode(job, "", isLast)
	}
	return output
}

// CountLines returns the number of lines the tree will render
func (et *ExecutionTree) CountLines() int {
	et.mu.Lock()
	defer et.mu.Unlock()

	count := 1 // root line
	for _, job := range et.Root.Children {
		count += countNodeLines(job)
	}
	return count
}

func countNodeLines(node *TreeNode) int {
	count := 1 // this node
	for _, child := range node.Children {
		count += countNodeLines(child)
	}
	return count
}

func renderNode(node *TreeNode, prefix string, isLast bool) string {
	output := ""

	// Determine branch character
	branch := "├─ "
	if isLast {
		branch = "└─ "
	}

	// Determine status indicator and color
	var status string
	var nameColor string
	switch node.Status {
	case StatusPassed:
		status = colors.BrightGreen("✓")
		nameColor = colors.BrightWhite(node.Name)
	case StatusFailed:
		status = colors.BrightRed("✗")
		nameColor = colors.BrightRed(node.Name)
	case StatusRunning:
		nameColor = colors.BrightYellow(node.Name)
		if node.Spinner != "" {
			status = node.Spinner
		} else {
			status = ""
		}
	default:
		// Pending/future items
		nameColor = colors.Gray(node.Name)
		status = ""
	}

	// Render this node
	output += prefix + branch + nameColor
	if status != "" {
		output += " " + status
	}
	output += "\n"

	// Render children
	if len(node.Children) > 0 {
		// Determine continuation character
		continuation := "│  "
		if isLast {
			continuation = "   "
		}

		for j, child := range node.Children {
			childIsLast := j == len(node.Children)-1
			output += renderNode(child, prefix+continuation, childIsLast)
		}
	}

	return output
}

// FinishPipeline marks the pipeline as passed or failed and renders final tree
func (et *ExecutionTree) FinishPipeline(passed bool, stepCount int) string {
	et.mu.Lock()
	defer et.mu.Unlock()

	if passed {
		et.Root.Status = StatusPassed
	} else {
		et.Root.Status = StatusFailed
	}

	output := colors.BrightGreen(et.Root.Name) + "\n"
	for i, job := range et.Root.Children {
		isLast := i == len(et.Root.Children)-1
		output += renderNode(job, "", isLast)
	}

	// Add final status line
	if passed {
		output += colors.BrightGreen(fmt.Sprintf("✓ PASS (%d steps passing)", stepCount)) + "\n"
	} else {
		output += colors.BrightRed("✗ FAIL") + "\n"
	}

	return output
}
