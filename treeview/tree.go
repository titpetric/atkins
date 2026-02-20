package treeview

// TreeNode represents a node in the execution tree (backward compatibility).
type TreeNode struct {
	*Node
}

// NewTreeNode creates a new tree node.
func NewTreeNode(name string) *TreeNode {
	return &TreeNode{
		Node: NewNode(name),
	}
}

// AddStep adds a step node to a job.
func (job *TreeNode) AddStep(stepName string) *TreeNode {
	job.Lock()
	defer job.Unlock()

	node := &TreeNode{
		Node: &Node{
			Name:   stepName,
			Status: StatusRunning,
		},
	}
	job.Children = append(job.Children, node.Node)
	return node
}

// AddStepDeferred adds a deferred step node to a job.
func (job *TreeNode) AddStepDeferred(stepName string) *TreeNode {
	job.Lock()
	defer job.Unlock()

	node := &TreeNode{
		Node: &Node{
			Name:     stepName,
			Status:   StatusRunning,
			Deferred: true,
		},
	}
	job.Children = append(job.Children, node.Node)
	return node
}

// SetStatus updates a node's status.
func (node *TreeNode) SetStatus(status Status) {
	node.Node.SetStatus(status)
}

// GetChildren returns the children of a node.
func (node *TreeNode) GetChildren() []*TreeNode {
	children := node.Node.GetChildren()
	result := make([]*TreeNode, len(children))
	for i, child := range children {
		result[i] = &TreeNode{Node: child}
	}
	return result
}

// GetName returns the name of the node.
func (node *TreeNode) GetName() string {
	return node.Name
}

// GetStatus returns the status of the node.
func (node *TreeNode) GetStatus() Status {
	return node.Status
}
