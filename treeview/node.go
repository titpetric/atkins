package treeview

import (
	"sync"
	"time"

	"github.com/titpetric/atkins/colors"
)

// Node represents a node in the tree (job, step, or iteration).
type Node struct {
	sync.Mutex

	Name         string
	ID           string // Unique identifier (e.g., "job.steps.0", "job.steps.1" for iterations)
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartOffset  float64 // Seconds offset from run start
	Duration     float64 // Duration in seconds
	If           string  // Condition that was evaluated (for conditional steps)
	Children     []*Node
	Dependencies []string
	Deferred     bool
	Summarize    bool
	Quiet        bool
	Output       []string // Multi-line output from command execution
}

// NewNode creates a new tree node.
func NewNode(name string) *Node {
	now := time.Now()
	return &Node{
		Name:         name,
		Status:       StatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
		Children:     make([]*Node, 0),
		Dependencies: make([]string, 0),
	}
}

// NewJobNode creates a new job node.
func NewJobNode(name string, nested bool) *Node {
	node := NewNode(name)
	if nested {
		node.Status = StatusConditional
	}
	return node
}

// NewStepNode creates a new step node.
func NewStepNode(name string, deferred bool) *Node {
	node := NewNode(name)
	node.Status = StatusRunning
	node.Deferred = deferred
	return node
}

// NewPendingStepNode creates a new step node with pending status.
func NewPendingStepNode(name string, deferred, summarize bool) *Node {
	node := NewNode(name)
	node.Status = StatusPending
	node.Deferred = deferred
	node.Summarize = summarize
	return node
}

// NewCmdNode creates a new command node as a child of a step.
func NewCmdNode(name string) *Node {
	return NewNode(name)
}

// StatusColor will return the status indicator for the node.
// The indicator contains ANSI color sequences. Thread-safe.
func (n *Node) StatusColor() string {
	n.Lock()
	defer n.Unlock()

	haveChildren := len(n.Children) > 0
	haveDeps := len(n.Dependencies) > 0

	status := n.Status.String()
	if status == "" && (haveChildren || haveDeps) {
		return colors.Green("●")
	}
	// For leaf nodes (no children, no deps), show a status indicator if in pending state
	if status == "" && !haveChildren && !haveDeps {
		return colors.Green("●")
	}
	return status
}

// Label returns the node label with color formatting. Thread-safe.
func (n *Node) Label() string {
	n.Lock()
	defer n.Unlock()

	haveChildren := len(n.Children) > 0
	haveDeps := len(n.Dependencies) > 0
	name := n.Name

	switch n.Status {
	case StatusRunning:
		if haveChildren {
			return colors.BrightOrange(name)
		}
		return colors.White(name)
	case StatusPassed:
		return colors.BrightWhite(name)
	case StatusFailed:
		return colors.BrightRed(name)
	case StatusSkipped:
		return colors.BrightYellow(name)
	case StatusConditional:
		return colors.BrightYellow(name)
	default:
		if haveChildren || haveDeps {
			return colors.BrightOrange(name)
		}
	}
	return colors.White(name)
}

// SetStatus updates a node's status thread-safely. Nil-safe: no-op on nil receiver.
func (n *Node) SetStatus(status Status) {
	if n == nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	n.Status = status
	n.Deferred = false
	n.UpdatedAt = time.Now()
}

// SetStartOffset sets the start offset from run start. Nil-safe: no-op on nil receiver.
func (n *Node) SetStartOffset(offset float64) {
	if n == nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	n.StartOffset = offset
}

// SetDuration sets the duration in seconds. Nil-safe: no-op on nil receiver.
func (n *Node) SetDuration(duration float64) {
	if n == nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	n.Duration = duration
	n.UpdatedAt = time.Now()
}

// SetIf sets the condition string that was evaluated. Nil-safe: no-op on nil receiver.
func (n *Node) SetIf(condition string) {
	if n == nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	n.If = condition
}

// SetSummarize sets the summarize flag. Nil-safe: no-op on nil receiver.
func (n *Node) SetSummarize(summarize bool) {
	if n == nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	n.Summarize = summarize
}

// SetID sets the node ID. Nil-safe: no-op on nil receiver.
func (n *Node) SetID(id string) {
	if n == nil {
		return
	}
	n.Lock()
	defer n.Unlock()
	n.ID = id
}

// SetOutput sets the output lines for this node (from command execution).
func (n *Node) SetOutput(lines []string) {
	n.Lock()
	defer n.Unlock()
	n.Output = lines
}

// GetOutput returns a copy of the output lines (thread-safe).
func (n *Node) GetOutput() []string {
	n.Lock()
	defer n.Unlock()
	if n.Output == nil {
		return nil
	}
	output := make([]string, len(n.Output))
	copy(output, n.Output)
	return output
}

// GetStatus returns the node's status (thread-safe).
func (n *Node) GetStatus() Status {
	n.Lock()
	defer n.Unlock()
	return n.Status
}

// GetIf returns the condition string (thread-safe).
func (n *Node) GetIf() string {
	n.Lock()
	defer n.Unlock()
	return n.If
}

// AddChild adds a child node.
func (n *Node) AddChild(child *Node) {
	n.Lock()
	defer n.Unlock()
	n.Children = append(n.Children, child)
}

// AddChildren adds multiple child nodes.
func (n *Node) AddChildren(children ...*Node) {
	n.Lock()
	defer n.Unlock()
	n.Children = append(n.Children, children...)
}

// HasChildren returns true or false if the node has children.
func (n *Node) HasChildren() bool {
	n.Lock()
	defer n.Unlock()

	return len(n.Children) > 0
}

// GetName returns the node name (thread-safe).
func (n *Node) GetName() string {
	n.Lock()
	defer n.Unlock()
	return n.Name
}

// GetDependencies returns a copy of the dependencies slice (thread-safe).
func (n *Node) GetDependencies() []string {
	n.Lock()
	defer n.Unlock()
	deps := make([]string, len(n.Dependencies))
	copy(deps, n.Dependencies)
	return deps
}

// IsSummarize returns the summarize flag (thread-safe).
func (n *Node) IsSummarize() bool {
	n.Lock()
	defer n.Unlock()
	return n.Summarize
}

// IsQuiet returns the quiet flag (thread-safe).
func (n *Node) IsQuiet() bool {
	n.Lock()
	defer n.Unlock()
	return n.Quiet
}

// GetChildren returns a copy of the children slice (thread-safe).
func (n *Node) GetChildren() []*Node {
	n.Lock()
	defer n.Unlock()
	children := make([]*Node, len(n.Children))
	copy(children, n.Children)
	return children
}
