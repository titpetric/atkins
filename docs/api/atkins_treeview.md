# Package ./treeview

```go
import (
	"github.com/titpetric/atkins/treeview"
)
```

Package `treeview` draws a pipeline run as a tree of jobs, steps and commands
on the terminal. Builder and ExecutionTree build the nodes from a pipeline,
Node holds the state the runner updates while the run goes on, Renderer
writes a tree as text and Trimmer fits the labels to the terminal width.
Display redraws the tree in place on a terminal and prints it once,
statically, on a pipe or a file.

## Types

<details>
<summary><code>type Builder</code></summary>

```go
// Builder constructs tree nodes from pipeline data.
type Builder struct {
	root *Node
}
```

</details>

<details>
<summary><code>type Display</code></summary>

```go
// Display renders the tree in place with ANSI cursor control. On a terminal
// each Render rolls back the lines of the previous one; Invalidate forgets
// them when output from outside the tree, such as an interactive command,
// lands between two renders. RenderFinal prints the whole tree once execution ends.
// The output stays scrollable. A silent display, for JSON and YAML output,
// prints nothing live.
type Display struct {
	lastLineCount int
	mu            sync.Mutex
	isTerminal    bool
	renderer      *Renderer
	finalOnly     bool
}
```

</details>

<details>
<summary><code>type ExecutionTree</code></summary>

```go
// ExecutionTree holds the entire execution tree.
type ExecutionTree struct {
	*TreeNode
}
```

</details>

<details>
<summary><code>type Node</code></summary>

```go
// Node is a job, step or iteration in the tree. Its accessors lock the node,
// the renderer reads it while the runner updates it, and its setters are
// no-ops on a nil node. A multi-line name keeps its first line with a
// summary of the rest, as TrimMultilineLabel writes it.
type Node struct {
	mu sync.Mutex

	Name         string
	ID           string // Unique identifier (e.g., "job.steps.0", "job.steps.1" for iterations)
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartOffset  float64 // Seconds offset from run start
	Duration     float64 // Duration in seconds
	If           string  // Condition the step evaluates (for conditional steps)
	Children     []*Node
	Dependencies []string
	Deferred     bool
	Summarize    bool
	Quiet        bool
	Output       []string // Multi-line output from command execution
}
```

</details>

<details>
<summary><code>type Renderer</code></summary>

```go
// Renderer handles rendering of tree nodes to strings with proper formatting.
type Renderer struct {
	mu        sync.Mutex
	trimmer   *Trimmer
	maxArgLen int
}
```

</details>

<details>
<summary><code>type Status</code></summary>

```go
// Status represents the execution status of a node.
type Status int
```

</details>

<details>
<summary><code>type TreeNode</code></summary>

```go
// TreeNode represents a node in the execution tree (backward compatibility).
type TreeNode struct {
	*Node
}
```

</details>

<details>
<summary><code>type Trimmer</code></summary>

```go
// Trimmer handles label and line trimming for viewport constraints.
type Trimmer struct {
	viewportWidth int
	mu            sync.RWMutex
}
```

</details>

## Consts

<details>
<summary><code>const DefaultMaxArgLen</code></summary>

```go
// DefaultMaxArgLen is the default length limit for argument values before compaction.
const DefaultMaxArgLen = 25
```

</details>

<details>
<summary><code>const StatusPending, StatusRunning, StatusPassed, StatusFailed, StatusSkipped, StatusConditional</code></summary>

```go
// Status constants.
const (
	StatusPending Status = iota
	StatusRunning
	StatusPassed
	StatusFailed
	StatusSkipped
	StatusConditional
)
```

</details>

## Function symbols

- `func BuildFromPipeline (pipeline *model.Pipeline, resolveDeps func(map[string]*model.Job, string) ([]string, error)) (*Node, error)`
- `func CompactArgList (args []string, maxWidth int) string`
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
- `func TrimMultilineLabel (label string) string`
- `func (*Builder) AddJob (job *model.Job, deps []string, jobName string) *TreeNode`
- `func (*Builder) AddJobWithSummary (job *model.Job, deps []string, jobName string) *TreeNode`
- `func (*Builder) AddJobWithoutSteps (deps []string, jobName string, nested bool) *TreeNode`
- `func (*Builder) Root () *Node`
- `func (*Display) Cleanup ()`
- `func (*Display) Invalidate ()`
- `func (*Display) IsTerminal () bool`
- `func (*Display) Render (root *Node)`
- `func (*Display) RenderFinal (root *Node)`
- `func (*Display) RenderStatic (root *Node)`
- `func (*ExecutionTree) AddJob (job *model.Job) *TreeNode`
- `func (*ExecutionTree) AddJobWithDeps (jobName string, deps []string) *TreeNode`
- `func (*ExecutionTree) CountLines () int`
- `func (*ExecutionTree) RenderTree () string`
- `func (*Node) AddChild (child *Node)`
- `func (*Node) AddChildren (children ...*Node)`
- `func (*Node) ClearChildren ()`
- `func (*Node) GetChildren () []*Node`
- `func (*Node) GetDependencies () []string`
- `func (*Node) GetIf () string`
- `func (*Node) GetName () string`
- `func (*Node) GetOutput () []string`
- `func (*Node) GetStatus () Status`
- `func (*Node) HasChildren () bool`
- `func (*Node) IsQuiet () bool`
- `func (*Node) IsSummarize () bool`
- `func (*Node) Label () string`
- `func (*Node) SetDuration (duration float64)`
- `func (*Node) SetID (id string)`
- `func (*Node) SetIf (condition string)`
- `func (*Node) SetName (name string)`
- `func (*Node) SetOutput (lines []string)`
- `func (*Node) SetQuiet (quiet bool)`
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

### `BuildFromPipeline`

`BuildFromPipeline` constructs a complete tree from a pipeline.
Returns the root node ready for rendering.

```go
func BuildFromPipeline(pipeline *model.Pipeline, resolveDeps func(map[string]*model.Job, string) ([]string, error)) (*Node, error)
```

### `CompactArgList`

`CompactArgList` truncates a space-separated argument list to fit within
maxWidth characters, replacing trailing arguments with <...N more>.

```go
func CompactArgList(args []string, maxWidth int) string
```

### `CompactArgs`

`CompactArgs` trims long argument values in a command string.
It replaces an argument longer than maxArgLen with <...N chars>.

```go
func CompactArgs(cmd string, maxArgLen int) string
```

### `CountLines`

`CountLines` returns the number of lines the tree will render.

```go
func CountLines(root *Node) int
```

### `NewBuilder`

`NewBuilder` creates a new tree builder.

```go
func NewBuilder(pipelineName string) *Builder
```

### `NewCmdNode`

`NewCmdNode` creates a new command node as a child of a step.

```go
func NewCmdNode(name string) *Node
```

### `NewDisplay`

`NewDisplay` creates a new display manager.

```go
func NewDisplay() *Display
```

### `NewDisplayWithFinal`

`NewDisplayWithFinal` creates a new display manager with final-only mode.

```go
func NewDisplayWithFinal(finalOnly bool) *Display
```

### `NewExecutionTree`

`NewExecutionTree` creates a new execution tree with a root node.

```go
func NewExecutionTree(pipelineName string) *ExecutionTree
```

### `NewJobNode`

`NewJobNode` creates a new job node.

```go
func NewJobNode(name string, nested bool) *Node
```

### `NewNode`

`NewNode` returns a pending node with the trimmed name.

```go
func NewNode(name string) *Node
```

### `NewPendingStepNode`

`NewPendingStepNode` creates a new step node with pending status.

```go
func NewPendingStepNode(name string, deferred, summarize bool) *Node
```

### `NewRenderer`

`NewRenderer` creates a new tree renderer.

```go
func NewRenderer() *Renderer
```

### `NewSilentDisplay`

`NewSilentDisplay` returns a display that renders nothing live.

```go
func NewSilentDisplay() *Display
```

### `NewStepNode`

`NewStepNode` creates a new step node.

```go
func NewStepNode(name string, deferred bool) *Node
```

### `NewTreeNode`

`NewTreeNode` creates a new tree node.

```go
func NewTreeNode(name string) *TreeNode
```

### `NewTrimmer`

`NewTrimmer` creates a new trimmer with detected viewport width.

```go
func NewTrimmer() *Trimmer
```

### `SortByOrder`

`SortByOrder` returns the set in orderList order, unlisted jobs last.

```go
func SortByOrder(jobSet map[string]bool, orderList []string) []string
```

### `SortJobsByDepth`

`SortJobsByDepth` returns the job names sorted by their count of ':', then by name.

```go
func SortJobsByDepth(jobNames []string) []string
```

### `TrimMultilineLabel`

`TrimMultilineLabel` trims a multi-line string to its first line,
appending a summary of remaining lines (e.g., "[... and 7 more]").

```go
func TrimMultilineLabel(label string) string
```

### `AddJob`

`AddJob` adds a job node to the tree with all its steps.

```go
func (*Builder) AddJob(job *model.Job, deps []string, jobName string) *TreeNode
```

### `AddJobWithSummary`

`AddJobWithSummary` adds a job node to the tree with summarization enabled.

```go
func (*Builder) AddJobWithSummary(job *model.Job, deps []string, jobName string) *TreeNode
```

### `AddJobWithoutSteps`

`AddJobWithoutSteps` adds a job node to the tree without steps (the caller adds steps afterwards).

```go
func (*Builder) AddJobWithoutSteps(deps []string, jobName string, nested bool) *TreeNode
```

### `Root`

`Root` returns the root node.

```go
func (*Builder) Root() *Node
```

### `Cleanup`

`Cleanup` is a no-op kept for API compatibility.

```go
func (*Display) Cleanup()
```

### `Invalidate`

`Invalidate` resets the line counter. The next Render draws below the current output.

```go
func (*Display) Invalidate()
```

### `IsTerminal`

`IsTerminal` returns whether stdout is a TTY.

```go
func (*Display) IsTerminal() bool
```

### `Render`

`Render` outputs the tree, updating in-place after the first render.

```go
func (*Display) Render(root *Node)
```

### `RenderFinal`

`RenderFinal` clears the live tree and prints the full tree statically.

```go
func (*Display) RenderFinal(root *Node)
```

### `RenderStatic`

`RenderStatic` displays a static tree view (for list).

```go
func (*Display) RenderStatic(root *Node)
```

### `AddJob`

`AddJob` adds a job node to the tree.

```go
func (*ExecutionTree) AddJob(job *model.Job) *TreeNode
```

### `AddJobWithDeps`

`AddJobWithDeps` adds a job node to the tree with dependencies.

```go
func (*ExecutionTree) AddJobWithDeps(jobName string, deps []string) *TreeNode
```

### `CountLines`

`CountLines` returns the number of lines the tree will render.

```go
func (*ExecutionTree) CountLines() int
```

### `RenderTree`

`RenderTree` renders the entire tree to a string (live rendering).

```go
func (*ExecutionTree) RenderTree() string
```

### `AddChild`

`AddChild` adds a child node.

```go
func (*Node) AddChild(child *Node)
```

### `AddChildren`

`AddChildren` adds child nodes.

```go
func (*Node) AddChildren(children ...*Node)
```

### `ClearChildren`

`ClearChildren` removes all child nodes.

```go
func (*Node) ClearChildren()
```

### `GetChildren`

`GetChildren` returns a copy of the children slice (thread-safe).

```go
func (*Node) GetChildren() []*Node
```

### `GetDependencies`

`GetDependencies` returns a copy of the dependencies slice (thread-safe).

```go
func (*Node) GetDependencies() []string
```

### `GetIf`

`GetIf` returns the condition string (thread-safe).

```go
func (*Node) GetIf() string
```

### `GetName`

`GetName` returns the node name (thread-safe).

```go
func (*Node) GetName() string
```

### `GetOutput`

`GetOutput` returns a copy of the output lines (thread-safe).

```go
func (*Node) GetOutput() []string
```

### `GetStatus`

`GetStatus` returns the node's status (thread-safe).

```go
func (*Node) GetStatus() Status
```

### `HasChildren`

`HasChildren` returns true or false if the node has children.

```go
func (*Node) HasChildren() bool
```

### `IsQuiet`

`IsQuiet` returns the quiet flag (thread-safe).

```go
func (*Node) IsQuiet() bool
```

### `IsSummarize`

`IsSummarize` returns the summarize flag (thread-safe).

```go
func (*Node) IsSummarize() bool
```

### `Label`

`Label` returns the node label with color formatting. Thread-safe.

```go
func (*Node) Label() string
```

### `SetDuration`

`SetDuration` sets the duration in seconds. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetDuration(duration float64)
```

### `SetID`

`SetID` sets the node ID. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetID(id string)
```

### `SetIf`

`SetIf` sets the evaluated condition string. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetIf(condition string)
```

### `SetName`

`SetName` returns early on a nil node.
SetName sets the trimmed display name.

```go
func (*Node) SetName(name string)
```

### `SetOutput`

`SetOutput` sets the output lines for this node (from command execution).

```go
func (*Node) SetOutput(lines []string)
```

### `SetQuiet`

`SetQuiet` sets the quiet flag. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetQuiet(quiet bool)
```

### `SetStartOffset`

`SetStartOffset` sets the start offset from run start. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetStartOffset(offset float64)
```

### `SetStatus`

`SetStatus` updates a node's status thread-safely. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetStatus(status Status)
```

### `SetSummarize`

`SetSummarize` sets the summarize flag. Nil-safe: no-op on nil receiver.

```go
func (*Node) SetSummarize(summarize bool)
```

### `StatusColor`

`StatusColor` will return the status indicator for the node.
The indicator contains ANSI color sequences. Thread-safe.

```go
func (*Node) StatusColor() string
```

### `Render`

`Render` converts a node to a string representation during execution (shows status for all nodes).

```go
func (*Renderer) Render(root *Node) string
```

### `RenderStatic`

`RenderStatic` renders a static tree (for list views) without spinners.

```go
func (*Renderer) RenderStatic(root *Node) string
```

### `AddStep`

`AddStep` adds a step node to a job.

```go
func (*TreeNode) AddStep(stepName string) *TreeNode
```

### `AddStepDeferred`

`AddStepDeferred` adds a deferred step node to a job.

```go
func (*TreeNode) AddStepDeferred(stepName string) *TreeNode
```

### `GetChildren`

`GetChildren` returns the children of a node.

```go
func (*TreeNode) GetChildren() []*TreeNode
```

### `GetName`

`GetName` returns the name of the node.

```go
func (*TreeNode) GetName() string
```

### `GetStatus`

`GetStatus` returns the status of the node.

```go
func (*TreeNode) GetStatus() Status
```

### `SetStatus`

`SetStatus` updates a node's status.

```go
func (*TreeNode) SetStatus(status Status)
```

### `GetViewportWidth`

`GetViewportWidth` returns the current viewport width.

```go
func (*Trimmer) GetViewportWidth() int
```

### `RefreshViewport`

`RefreshViewport` re-detects the terminal width (call before each render).

```go
func (*Trimmer) RefreshViewport()
```

### `SetViewportWidth`

`SetViewportWidth` sets a custom viewport width (useful for testing).

```go
func (*Trimmer) SetViewportWidth(width int)
```

### `TrimLabel`

`TrimLabel` returns the label with values over maxArgLen compacted, fitted to the viewport after prefixLen.

```go
func (*Trimmer) TrimLabel(label string, maxArgLen, prefixLen int) string
```

### `TrimToViewport`

`TrimToViewport` trims a line to fit within the viewport width.
It accounts for ANSI escape codes and adds "..." suffix when trimmed.
The prefixLen parameter indicates how many visual characters of prefix
(indentation, branch characters) are already used.

```go
func (*Trimmer) TrimToViewport(line string, prefixLen int) string
```

### `Label`

`Label` returns a lowercase readable label for the Status (for logging/serialization).

```go
func (Status) Label() string
```

### `String`

`String` returns a colored string representation of the Status for display.

```go
func (Status) String() string
```
