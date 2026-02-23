# Package ./treeview

```go
import (
	"github.com/titpetric/atkins/treeview"
}
```

## Types

```go
// Builder constructs tree nodes from pipeline data.
type Builder struct {
	root *Node
}
```

```go
// Display manages in-place tree rendering with ANSI cursor control.
type Display struct {
	lastLineCount	int
	mu		sync.Mutex
	isTerminal	bool
	renderer	*Renderer
	finalOnly	bool
}
```

```go
// ExecutionTree holds the entire execution tree.
type ExecutionTree struct {
	*TreeNode
}
```

```go
// Node represents a node in the tree (job, step, or iteration).
type Node struct {
	sync.Mutex

	Name		string
	ID		string	// Unique identifier (e.g., "job.steps.0", "job.steps.1" for iterations)
	Status		Status
	CreatedAt	time.Time
	UpdatedAt	time.Time
	StartOffset	float64	// Seconds offset from run start
	Duration	float64	// Duration in seconds
	If		string	// Condition that was evaluated (for conditional steps)
	Children	[]*Node
	Dependencies	[]string
	Deferred	bool
	Summarize	bool
	Output		[]string	// Multi-line output from command execution
}
```

```go
// Renderer handles rendering of tree nodes to strings with proper formatting.
type Renderer struct {
	mu		sync.Mutex
	trimmer		*Trimmer
	maxArgLen	int
}
```

```go
// Status represents the execution status of a node.
type Status int
```

```go
// TreeNode represents a node in the execution tree (backward compatibility).
type TreeNode struct {
	*Node
}
```

```go
// Trimmer handles label and line trimming for viewport constraints.
type Trimmer struct {
	viewportWidth	int
	mu		sync.RWMutex
}
```

## Consts

```go
// DefaultMaxArgLen is the default maximum length for argument values before compaction.
const DefaultMaxArgLen = 25
```

```go
// Status constants.
const (
	StatusPending	Status	= iota
	StatusRunning
	StatusPassed
	StatusFailed
	StatusSkipped
	StatusConditional
)
```

## Function symbols

- `func BuildFromPipeline (pipeline *model.Pipeline, resolveDeps func(map[string]*model.Job, string) ([]string, error)) (*Node, error)`
- `func CompactArgs (cmd string, maxArgLen int) string`
- `func CountLines (root *Node) int`
- `func NewBuilder (pipelineName string) *Builder`
- `func NewCmdNode (name string) *Node`
- `func NewDisplay () *Display`
- `func NewDisplayWithFinal (finalOnly bool) *Display`
- `func NewExecutionTree (pipelineName string) *ExecutionTree`
- `func NewJobNode (name string, nested bool) *Node`
- `func NewNode (name string) *Node`
- `func NewPendingStepNode (name string, deferred,summarize bool) *Node`
- `func NewRenderer () *Renderer`
- `func NewSilentDisplay () *Display`
- `func NewStepNode (name string, deferred bool) *Node`
- `func NewTreeNode (name string) *TreeNode`
- `func NewTrimmer () *Trimmer`
- `func SortByOrder (jobSet map[string]bool, orderList []string) []string`
- `func SortJobsByDepth (jobNames []string) []string`
- `func (*Builder) AddJob (job *model.Job, deps []string, jobName string) *TreeNode`
- `func (*Builder) AddJobWithSummary (job *model.Job, deps []string, jobName string) *TreeNode`
- `func (*Builder) AddJobWithoutSteps (deps []string, jobName string, nested bool) *TreeNode`
- `func (*Builder) Root () *Node`
- `func (*Display) IsTerminal () bool`
- `func (*Display) Render (root *Node)`
- `func (*Display) RenderStatic (root *Node)`
- `func (*ExecutionTree) AddJob (job *model.Job) *TreeNode`
- `func (*ExecutionTree) AddJobWithDeps (jobName string, deps []string) *TreeNode`
- `func (*ExecutionTree) CountLines () int`
- `func (*ExecutionTree) RenderTree () string`
- `func (*Node) AddChild (child *Node)`
- `func (*Node) AddChildren (children ...*Node)`
- `func (*Node) GetChildren () []*Node`
- `func (*Node) HasChildren () bool`
- `func (*Node) Label () string`
- `func (*Node) SetDuration (duration float64)`
- `func (*Node) SetID (id string)`
- `func (*Node) SetIf (condition string)`
- `func (*Node) SetOutput (lines []string)`
- `func (*Node) SetStartOffset (offset float64)`
- `func (*Node) SetStatus (status Status)`
- `func (*Node) SetSummarize (summarize bool)`
- `func (*Node) StatusColor () string`
- `func (*Renderer) Render (root *Node) string`
- `func (*Renderer) RenderStatic (root *Node) string`
- `func (*TreeNode) AddStep (stepName string) *TreeNode`
- `func (*TreeNode) AddStepDeferred (stepName string) *TreeNode`
- `func (*TreeNode) GetChildren () []*TreeNode`
- `func (*TreeNode) GetName () string`
- `func (*TreeNode) GetStatus () Status`
- `func (*TreeNode) SetStatus (status Status)`
- `func (*Trimmer) GetViewportWidth () int`
- `func (*Trimmer) RefreshViewport ()`
- `func (*Trimmer) SetViewportWidth (width int)`
- `func (*Trimmer) TrimLabel (label string, maxArgLen,prefixLen int) string`
- `func (*Trimmer) TrimToViewport (line string, prefixLen int) string`
- `func (Status) Label () string`
- `func (Status) String () string`

### BuildFromPipeline

BuildFromPipeline constructs a complete tree from a pipeline.
Returns the root node ready to be rendered.

```go
func BuildFromPipeline (pipeline *model.Pipeline, resolveDeps func(map[string]*model.Job, string) ([]string, error)) (*Node, error)
```

### CompactArgs

CompactArgs trims long argument values in a command string.
Arguments longer than maxArgLen are replaced with <...N chars>.

```go
func CompactArgs (cmd string, maxArgLen int) string
```

### CountLines

CountLines returns the number of lines the tree will render.

```go
func CountLines (root *Node) int
```

### NewBuilder

NewBuilder creates a new tree builder.

```go
func NewBuilder (pipelineName string) *Builder
```

### NewCmdNode

NewCmdNode creates a new command node as a child of a step.

```go
func NewCmdNode (name string) *Node
```

### NewDisplay

NewDisplay creates a new display manager.

```go
func NewDisplay () *Display
```

### NewDisplayWithFinal

NewDisplayWithFinal creates a new display manager with final-only mode.

```go
func NewDisplayWithFinal (finalOnly bool) *Display
```

### NewExecutionTree

NewExecutionTree creates a new execution tree with a root node.

```go
func NewExecutionTree (pipelineName string) *ExecutionTree
```

### NewJobNode

NewJobNode creates a new job node.

```go
func NewJobNode (name string, nested bool) *Node
```

### NewNode

NewNode creates a new tree node.

```go
func NewNode (name string) *Node
```

### NewPendingStepNode

NewPendingStepNode creates a new step node with pending status.

```go
func NewPendingStepNode (name string, deferred,summarize bool) *Node
```

### NewRenderer

NewRenderer creates a new tree renderer.

```go
func NewRenderer () *Renderer
```

### NewSilentDisplay

NewSilentDisplay creates a display that produces no output.
Used when JSON/YAML output mode is enabled.

```go
func NewSilentDisplay () *Display
```

### NewStepNode

NewStepNode creates a new step node.

```go
func NewStepNode (name string, deferred bool) *Node
```

### NewTreeNode

NewTreeNode creates a new tree node.

```go
func NewTreeNode (name string) *TreeNode
```

### NewTrimmer

NewTrimmer creates a new trimmer with detected viewport width.

```go
func NewTrimmer () *Trimmer
```

### SortByOrder

SortByOrder returns the job names from the set in the order specified by orderList.
Jobs in the set that are not in orderList are appended at the end.

```go
func SortByOrder (jobSet map[string]bool, orderList []string) []string
```

### SortJobsByDepth

SortJobsByDepth sorts job names by ':' depth, then alphabetically.
Depth is determined by the count of ':' separators in the job name.

```go
func SortJobsByDepth (jobNames []string) []string
```

### AddJob

AddJob adds a job node to the tree with all its steps.

```go
func (*Builder) AddJob (job *model.Job, deps []string, jobName string) *TreeNode
```

### AddJobWithSummary

AddJobWithSummary adds a job node to the tree with summarization enabled.

```go
func (*Builder) AddJobWithSummary (job *model.Job, deps []string, jobName string) *TreeNode
```

### AddJobWithoutSteps

AddJobWithoutSteps adds a job node to the tree without steps (steps should be added manually afterwards).

```go
func (*Builder) AddJobWithoutSteps (deps []string, jobName string, nested bool) *TreeNode
```

### Root

Root returns the root node.

```go
func (*Builder) Root () *Node
```

### IsTerminal

IsTerminal returns whether stdout is a TTY.

```go
func (*Display) IsTerminal () bool
```

### Render

Render outputs the tree, updating in-place if previously rendered.

```go
func (*Display) Render (root *Node)
```

### RenderStatic

RenderStatic displays a static tree view (for list).

```go
func (*Display) RenderStatic (root *Node)
```

### AddJob

AddJob adds a job node to the tree.

```go
func (*ExecutionTree) AddJob (job *model.Job) *TreeNode
```

### AddJobWithDeps

AddJobWithDeps adds a job node to the tree with dependencies.

```go
func (*ExecutionTree) AddJobWithDeps (jobName string, deps []string) *TreeNode
```

### CountLines

CountLines returns the number of lines the tree will render.

```go
func (*ExecutionTree) CountLines () int
```

### RenderTree

RenderTree renders the entire tree to a string (live rendering).

```go
func (*ExecutionTree) RenderTree () string
```

### AddChild

AddChild adds a child node.

```go
func (*Node) AddChild (child *Node)
```

### AddChildren

AddChildren adds multiple child nodes.

```go
func (*Node) AddChildren (children ...*Node)
```

### GetChildren

GetChildren returns a copy of the children slice (thread-safe).

```go
func (*Node) GetChildren () []*Node
```

### HasChildren

HasChildren returns true or false if the node has children.

```go
func (*Node) HasChildren () bool
```

### SetDuration

SetDuration sets the duration in seconds. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetDuration (duration float64)
```

### SetID

SetID sets the node ID. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetID (id string)
```

### SetIf

SetIf sets the condition string that was evaluated. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetIf (condition string)
```

### SetOutput

SetOutput sets the output lines for this node (from command execution).

```go
func (*Node) SetOutput (lines []string)
```

### SetStartOffset

SetStartOffset sets the start offset from run start. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetStartOffset (offset float64)
```

### SetStatus

SetStatus updates a node's status thread-safely. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetStatus (status Status)
```

### SetSummarize

SetSummarize sets the summarize flag. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetSummarize (summarize bool)
```

### StatusColor

StatusColor will return the status indicator for the node.
The indicator contains ANSI color sequences.

```go
func (*Node) StatusColor () string
```

### Render

Render converts a node to a string representation during execution (shows status for all nodes).

```go
func (*Renderer) Render (root *Node) string
```

### RenderStatic

RenderStatic renders a static tree (for list views) without spinners.

```go
func (*Renderer) RenderStatic (root *Node) string
```

### AddStep

AddStep adds a step node to a job.

```go
func (*TreeNode) AddStep (stepName string) *TreeNode
```

### AddStepDeferred

AddStepDeferred adds a deferred step node to a job.

```go
func (*TreeNode) AddStepDeferred (stepName string) *TreeNode
```

### GetChildren

GetChildren returns the children of a node.

```go
func (*TreeNode) GetChildren () []*TreeNode
```

### GetName

GetName returns the name of the node.

```go
func (*TreeNode) GetName () string
```

### GetStatus

GetStatus returns the status of the node.

```go
func (*TreeNode) GetStatus () Status
```

### SetStatus

SetStatus updates a node's status.

```go
func (*TreeNode) SetStatus (status Status)
```

### GetViewportWidth

GetViewportWidth returns the current viewport width.

```go
func (*Trimmer) GetViewportWidth () int
```

### RefreshViewport

RefreshViewport re-detects the terminal width (call before each render).

```go
func (*Trimmer) RefreshViewport ()
```

### SetViewportWidth

SetViewportWidth sets a custom viewport width (useful for testing).

```go
func (*Trimmer) SetViewportWidth (width int)
```

### TrimLabel

TrimLabel applies both compaction and viewport trimming to a label.
- maxArgLen: maximum length for argument values before compaction
- prefixLen: visual length of the prefix (indentation, branch chars).

```go
func (*Trimmer) TrimLabel (label string, maxArgLen,prefixLen int) string
```

### TrimToViewport

TrimToViewport trims a line to fit within the viewport width.
It accounts for ANSI escape codes and adds "..." suffix when trimmed.
The prefixLen parameter indicates how many visual characters of prefix
(indentation, branch characters) are already used.

```go
func (*Trimmer) TrimToViewport (line string, prefixLen int) string
```

### Label

Label returns a lowercase readable label for the Status (for logging/serialization).

```go
func (Status) Label () string
```

### String

String returns a colored string representation of the Status for display.

```go
func (Status) String () string
```

### Label

```go
func (*Node) Label () string
```


