# Package ./model

```go
import (
	"github.com/titpetric/atkins/model"
)
```

Package `model` holds the types a pipeline file decodes into: Pipeline, Job,
Step and the declarations they share (Decl: vars, env and include). The
types accept both the GitHub Actions and the Taskfile spellings of a field,
and their UnmarshalYAML methods take the shorthand forms, such as a job or
a step written as one command string.

The package describes a pipeline and runs nothing. Walk and Dispatch visit
jobs and steps in a fixed order for the runner and the linter, which
decide what a visit does. types.mig.go holds the models of the event
database, which mig generates from schema/*.up.sql.

## Types

<details>
<summary><code>type Command</code></summary>

```go
// Command generated for db table `command`.
type Command struct {
	// ID
	ID string `db:"id" json:"id"`

	// Invocation ID
	InvocationID string `db:"invocation_id" json:"invocation_id"`

	// Parent ID
	ParentID string `db:"parent_id" json:"parent_id"`

	// Kind
	Kind string `db:"kind" json:"kind"`

	// Command
	Command string `db:"command" json:"command"`

	// Normalized
	Normalized string `db:"normalized" json:"normalized"`

	// Tool
	Tool string `db:"tool" json:"tool"`

	// Dir
	Dir string `db:"dir" json:"dir"`

	// Exit Code
	ExitCode int64 `db:"exit_code" json:"exit_code"`

	// Error
	Error string `db:"error" json:"error"`

	// Duration Ms
	DurationMs int64 `db:"duration_ms" json:"duration_ms"`

	// Started At
	StartedAt *time.Time `db:"started_at" json:"started_at"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`
}
```

</details>

<details>
<summary><code>type Condition</code></summary>

```go
// Condition represents a single if-condition expression.
type Condition string
```

</details>

<details>
<summary><code>type Conditionals</code></summary>

```go
// Conditionals is a slice of Condition with custom YAML unmarshalling
// to support both single string and list of strings.
type Conditionals []Condition
```

</details>

<details>
<summary><code>type Decl</code></summary>

```go
// Decl represents a common variables signature {vars, env, include}.
// It's a base type for pipelines, jobs/tasks and steps/cmds.
type Decl struct {
	Vars    map[string]any `yaml:"vars,omitempty"`
	Include *IncludeDecl   `yaml:"include,omitempty"`
	Env     *EnvDecl       `yaml:"env,omitempty"`
}
```

</details>

<details>
<summary><code>type DeferredStep</code></summary>

```go
// DeferredStep represents a deferred step wrapper.
type DeferredStep struct {
	Defer *Step `yaml:"defer,omitempty"`
}
```

</details>

<details>
<summary><code>type Dependencies</code></summary>

```go
// Dependencies represents job dependencies.
type Dependencies []string
```

</details>

<details>
<summary><code>type EnvDecl</code></summary>

```go
// EnvDecl represents an environment variable declaration that can contain
// both manually-set variables and includes from external files.
type EnvDecl Decl
```

</details>

<details>
<summary><code>type IncludeDecl</code></summary>

```go
// IncludeDecl represents file includes that can be either a single string or a list of strings.
type IncludeDecl struct {
	Files []string
}
```

</details>

<details>
<summary><code>type Iterator</code></summary>

```go
// Iterator represents a single for-loop iteration specification,
// using format "item in items" or "(idx, item) in items". The alternative
// "items as item" syntax decodes to the normalized "item in items".
type Iterator string
```

</details>

<details>
<summary><code>type Iterators</code></summary>

```go
// Iterators is a slice of Iterator with custom YAML unmarshalling
// to support both single string and list of strings.
type Iterators []Iterator
```

</details>

<details>
<summary><code>type Job</code></summary>

```go
// Job represents a job/task in the pipeline.
type Job struct {
	*Decl

	Desc        string       `yaml:"desc,omitempty"`
	Dir         string       `yaml:"dir,omitempty"`
	If          Conditionals `yaml:"if,omitempty"`
	For         Iterators    `yaml:"for,omitempty"`
	Cmd         string       `yaml:"cmd,omitempty"`
	Cmds        []*Step      `yaml:"cmds,omitempty"`
	Run         string       `yaml:"run,omitempty"`
	Steps       []*Step      `yaml:"steps,omitempty"`
	Detach      bool         `yaml:"detach,omitempty"`
	Show        *bool        `yaml:"show,omitempty"` // Show in display (true=show, false=hide, nil=show if root level/ invoked)
	DependsOn   Dependencies `yaml:"depends_on,omitempty"`
	Aliases     []string     `yaml:"aliases,omitempty"`  // Alternative names for invoking this job
	Requires    []string     `yaml:"requires,omitempty"` // Variables required when invoked in a loop
	Timeout     string       `yaml:"timeout,omitempty"`  // e.g., "10m", "300s"
	Summarize   bool         `yaml:"summarize,omitempty"`
	Quiet       bool         `yaml:"quiet,omitempty"`
	Passthru    bool         `yaml:"passthru,omitempty"`    // If true, print output with tree indentation
	Interactive bool         `yaml:"interactive,omitempty"` // If true, stream output live and connect stdin for keyboard input

	// Args lets the command line pass arguments to the job: everything
	// after a bare -- goes to the job's exec: steps as literal argv.
	Args bool `yaml:"args,omitempty"`

	// When gates the job on files in the folder atkins runs from, with
	// the same rules as a skill's when: block.
	When *PipelineWhen `yaml:"when,omitempty"`

	Name   string `yaml:"-"`
	Nested bool   `yaml:"-"`

	// Inactive marks a job whose when: block does not match. The job
	// stays out of the listing, and running it fails with an error.
	Inactive bool `yaml:"-"`

	// Matched is the file that satisfied the job's when: block.
	Matched string `yaml:"-"`
}
```

</details>

<details>
<summary><code>type JobWalkFunc</code></summary>

```go
// JobWalkFunc runs for each step in a job during Walk.
type JobWalkFunc func(index int, step *Step) error
```

</details>

<details>
<summary><code>type Label</code></summary>

```go
// Label is the display label of a step or command. NewLabel starts one and
// the With* builders customize it. String is the plain name used for node
// names and comparisons; ForDisplay adds the colour and status for rendering.
type Label struct {
	Text       string              // The display text (e.g., "docker compose up" or "run: splint fix ./...")
	Type       string              // The operation kind: "task", "run", "cmd", "cmds"
	ShowPrefix bool                // Whether to display the type prefix (e.g., "run:")
	Status     string              // Optional status indicator (e.g., "●", "✓", "✗")
	Color      func(string) string // Optional color function to apply to the label
}
```

</details>

<details>
<summary><code>type Migrations</code></summary>

```go
// Migrations generated for db table `migrations`.
type Migrations struct {
	// Project
	Project string `db:"project" json:"project"`

	// Filename
	Filename string `db:"filename" json:"filename"`

	// Statement Index
	StatementIndex int64 `db:"statement_index" json:"statement_index"`

	// Status
	Status string `db:"status" json:"status"`
}
```

</details>

<details>
<summary><code>type Pipeline</code></summary>

```go
// Pipeline represents the root structure of an atkins.yml file.
type Pipeline struct {
	*Decl

	ID   string `yaml:"-"`
	Name string `yaml:"name,omitempty"`
	Dir  string `yaml:"dir,omitempty"`

	// Help is one or two sentences on what the pipeline is for, and the
	// files it reads and writes. `atkins --help` prints it under the
	// skill the pipeline came from.
	Help string `yaml:"help,omitempty"`

	Jobs  map[string]*Job `yaml:"jobs,omitempty"`
	Tasks map[string]*Job `yaml:"tasks,omitempty"`

	When *PipelineWhen `yaml:"when,omitempty"`

	// Path is the file the pipeline loads from, empty for stdin.
	Path string `yaml:"-"`

	// Matched is the file that satisfied the when: block of a skill,
	// empty when the skill has no when: block or it does not match.
	Matched string `yaml:"-"`

	// Inactive marks a skill whose when: block does not match. Only the
	// listing of -l --all carries one; it contributes no jobs to a run.
	Inactive bool `yaml:"-"`
}
```

</details>

<details>
<summary><code>type PipelineWalkFunc</code></summary>

```go
// PipelineWalkFunc runs for each job in a pipeline during Walk.
type PipelineWalkFunc func(name string, job *Job) error
```

</details>

<details>
<summary><code>type PipelineWhen</code></summary>

```go
// PipelineWhen is a list of files that need to exist somewhere to
// enable the pipeline, e.g. compose.yml for compose pipeline.
//
// A job carries the same block to gate one job of a pipeline: where
// its when: block does not match, the job stays out of the listing and
// running it fails with an error.
type PipelineWhen struct {
	// Files lists the patterns, any of which activates the block. Each
	// pattern resolves in the starting folder and then in every parent
	// folder, nearest first.
	Files []string `yaml:"files" json:"files"`

	// Here restricts the search to the starting folder. A match in
	// a parent folder does not count.
	Here bool `yaml:"here,omitempty" json:"here,omitempty"`

	// Contains is a regular expression the matched file must match.
	// With Contains, a pattern resolving to a folder never matches.
	Contains string `yaml:"contains,omitempty" json:"contains,omitempty"`
}
```

</details>

<details>
<summary><code>type QueryConfig</code></summary>

```go
// QueryConfig is a function-chaining SQL statement type.
type QueryConfig struct {
	Table       string
	Columns     []string
	Where       string
	OrderBy     string
	LimitStart  int
	LimitOffset int
	Statement   string
}
```

</details>

<details>
<summary><code>type QueryOption</code></summary>

```go
// QueryOption is implemented by each data model type.
type QueryOption interface {
	WithTable(name string) QueryOption
	WithColumns(cols []string) QueryOption
	WithWhere(clause string) QueryOption
	WithOrderBy(clause string) QueryOption
	WithLimit(start, offset int) QueryOption
	WithStatement(stmt string) QueryOption
}
```

</details>

<details>
<summary><code>type ResolvedTask</code></summary>

```go
// ResolvedTask contains the result of resolving a task reference.
type ResolvedTask struct {
	Name     string    // Canonical name (e.g., "go:build" or "build")
	Job      *Job      // The resolved job
	Pipeline *Pipeline // The pipeline containing the task
}
```

</details>

<details>
<summary><code>type Run</code></summary>

```go
// Run generated for db table `run`.
type Run struct {
	// ID
	ID string `db:"id" json:"id"`

	// Invocation ID
	InvocationID string `db:"invocation_id" json:"invocation_id"`

	// Parent ID
	ParentID string `db:"parent_id" json:"parent_id"`

	// Kind
	Kind string `db:"kind" json:"kind"`

	// Name
	Name string `db:"name" json:"name"`

	// Pipeline
	Pipeline string `db:"pipeline" json:"pipeline"`

	// File
	File string `db:"file" json:"file"`

	// Dir
	Dir string `db:"dir" json:"dir"`

	// Module
	Module string `db:"module" json:"module"`

	// Repository
	Repository string `db:"repository" json:"repository"`

	// Branch
	Branch string `db:"branch" json:"branch"`

	// Commit Sha
	CommitSha string `db:"commit_sha" json:"commit_sha"`

	// Result
	Result string `db:"result" json:"result"`

	// Error
	Error string `db:"error" json:"error"`

	// Steps
	Steps int64 `db:"steps" json:"steps"`

	// Duration Ms
	DurationMs int64 `db:"duration_ms" json:"duration_ms"`

	// Started At
	StartedAt *time.Time `db:"started_at" json:"started_at"`

	// Finished At
	FinishedAt *time.Time `db:"finished_at" json:"finished_at"`

	// Created At
	CreatedAt *time.Time `db:"created_at" json:"created_at"`
}
```

</details>

<details>
<summary><code>type Step</code></summary>

```go
// Step is one step of a job: a task: step runs other jobs, any other step
// runs commands (run:, cmd:, cmds: or exec:). A task: value may name more
// than one job, separated by whitespace (`task: lint test build`), which run
// in sequence and match one task: step per job. An exec: argv runs without a shell; its
// display string quotes an argument holding a space or a shell character.
type Step struct {
	*Decl

	Name        string       `yaml:"name,omitempty"`
	Desc        string       `yaml:"desc,omitempty"`
	Dir         string       `yaml:"dir,omitempty"`
	Run         string       `yaml:"run,omitempty"`
	Cmd         string       `yaml:"cmd,omitempty"`
	Cmds        []string     `yaml:"cmds,omitempty"`
	Exec        []string     `yaml:"exec,omitempty"` // Program and arguments, run without a shell
	Task        string       `yaml:"task,omitempty"` // Task/job name to invoke
	If          Conditionals `yaml:"if,omitempty"`
	For         Iterators    `yaml:"for,omitempty"`
	Detach      bool         `yaml:"detach,omitempty"`
	Deferred    bool         `yaml:"deferred,omitempty"`
	Verbose     bool         `yaml:"verbose,omitempty"`
	Summarize   bool         `yaml:"summarize,omitempty"`
	Quiet       bool         `yaml:"quiet,omitempty"`
	Passthru    bool         `yaml:"passthru,omitempty"`    // If true, print output with tree indentation
	Interactive bool         `yaml:"interactive,omitempty"` // If true, stream output live and connect stdin for keyboard input
	HidePrefix  bool         `yaml:"-"`                     // If true, don't show "run:" prefix in display
}
```

</details>

<details>
<summary><code>type StepDispatcher</code></summary>

```go
// StepDispatcher handles dispatching step execution without caring
// about the step type (task vs command). Implementations decide how
// to execute each type.
type StepDispatcher interface {
	// Task runs when the step references another job via task:.
	Task(step *Step) error
	// Command runs for each executable command in the step.
	Command(step *Step, index int, cmd string) error
}
```

</details>

<details>
<summary><code>type VariableStorage</code></summary>

```go
// VariableStorage stores variables with lazy evaluation support.
// Implementations may defer evaluation until the first Get call.
type VariableStorage interface {
	// Get returns a variable value. If the variable is pending evaluation,
	// Get evaluates it on first access and caches the result.
	Get(key string) any

	// Set stores a value directly (already evaluated).
	// Used for loop variables and runtime-computed values.
	Set(key string, value any)

	// Clone creates an independent copy for iteration scopes.
	Clone() VariableStorage

	// Walk iterates over all evaluated variables.
	// Walk calls the function for each key-value pair while holding a lock.
	Walk(fn func(key string, value any))
}
```

</details>

## Consts

<details>
<summary><code>const CommandTable</code></summary>

```go
// CommandTable is the name of the table in the DB.
const CommandTable = "`command`"
```

</details>

<details>
<summary><code>const MigrationsTable</code></summary>

```go
// MigrationsTable is the name of the table in the DB.
const MigrationsTable = "`migrations`"
```

</details>

<details>
<summary><code>const RunTable</code></summary>

```go
// RunTable is the name of the table in the DB.
const RunTable = "`run`"
```

</details>

## Vars

<details>
<summary><code>var CommandFields</code></summary>

```go
// CommandFields is a list of all columns in the DB table.
var CommandFields = []string{"id", "invocation_id", "parent_id", "kind", "command", "normalized", "tool", "dir", "exit_code", "error", "duration_ms", "started_at", "created_at"}
```

</details>

<details>
<summary><code>var CommandPrimaryFields</code></summary>

```go
// CommandPrimaryFields are the primary key fields in the DB table.
var CommandPrimaryFields = []string{"id"}
```

</details>

<details>
<summary><code>var MigrationsFields</code></summary>

```go
// MigrationsFields is a list of all columns in the DB table.
var MigrationsFields = []string{"project", "filename", "statement_index", "status"}
```

</details>

<details>
<summary><code>var MigrationsPrimaryFields</code></summary>

```go
// MigrationsPrimaryFields are the primary key fields in the DB table.
var MigrationsPrimaryFields = []string{"project", "filename"}
```

</details>

<details>
<summary><code>var RunFields</code></summary>

```go
// RunFields is a list of all columns in the DB table.
var RunFields = []string{"id", "invocation_id", "parent_id", "kind", "name", "pipeline", "file", "dir", "module", "repository", "branch", "commit_sha", "result", "error", "steps", "duration_ms", "started_at", "finished_at", "created_at"}
```

</details>

<details>
<summary><code>var RunPrimaryFields</code></summary>

```go
// RunPrimaryFields are the primary key fields in the DB table.
var RunPrimaryFields = []string{"id"}
```

</details>

## Function symbols

- `func NewLabel (text,labelType string) *Label`
- `func NewResolvedTask (pipeline *Pipeline, job *Job, name string) *ResolvedTask`
- `func WithColumns (cols []string) QueryOption`
- `func WithLimit (start,offset int) QueryOption`
- `func WithOrderBy (clause string) QueryOption`
- `func WithStatement (stmt string) QueryOption`
- `func WithTable (name string) QueryOption`
- `func WithWhere (clause string) QueryOption`
- `func (*Command) Delete (opts ...QueryOption) string`
- `func (*Command) GetCommand () string`
- `func (*Command) GetCreatedAt () *time.Time`
- `func (*Command) GetDir () string`
- `func (*Command) GetDurationMs () int64`
- `func (*Command) GetError () string`
- `func (*Command) GetExitCode () int64`
- `func (*Command) GetID () string`
- `func (*Command) GetInvocationID () string`
- `func (*Command) GetKind () string`
- `func (*Command) GetNormalized () string`
- `func (*Command) GetParentID () string`
- `func (*Command) GetStartedAt () *time.Time`
- `func (*Command) GetTool () string`
- `func (*Command) Insert (opts ...QueryOption) string`
- `func (*Command) Select (opts ...QueryOption) string`
- `func (*Command) SetCommand (val string)`
- `func (*Command) SetCreatedAt (stamp time.Time)`
- `func (*Command) SetDir (val string)`
- `func (*Command) SetDurationMs (val int64)`
- `func (*Command) SetError (val string)`
- `func (*Command) SetExitCode (val int64)`
- `func (*Command) SetID (val string)`
- `func (*Command) SetInvocationID (val string)`
- `func (*Command) SetKind (val string)`
- `func (*Command) SetNormalized (val string)`
- `func (*Command) SetParentID (val string)`
- `func (*Command) SetStartedAt (stamp time.Time)`
- `func (*Command) SetTool (val string)`
- `func (*Command) Update (opts ...QueryOption) string`
- `func (*Conditionals) UnmarshalYAML (node *yaml.Node) error`
- `func (*Dependencies) UnmarshalYAML (node *yaml.Node) error`
- `func (*IncludeDecl) UnmarshalYAML (node *yaml.Node) error`
- `func (*Iterators) UnmarshalYAML (node *yaml.Node) error`
- `func (*Job) Children () []*Step`
- `func (*Job) GetAliases () []string`
- `func (*Job) IsRootLevel () bool`
- `func (*Job) ShouldShow () bool`
- `func (*Job) UnmarshalYAML (node *yaml.Node) error`
- `func (*Job) Walk (fn JobWalkFunc) error`
- `func (*Label) ForDisplay () string`
- `func (*Label) String () string`
- `func (*Label) WithColor (colorFn func(string) string) *Label`
- `func (*Label) WithPrefix (show bool) *Label`
- `func (*Label) WithStatus (status string) *Label`
- `func (*Migrations) Delete (opts ...QueryOption) string`
- `func (*Migrations) GetFilename () string`
- `func (*Migrations) GetProject () string`
- `func (*Migrations) GetStatementIndex () int64`
- `func (*Migrations) GetStatus () string`
- `func (*Migrations) Insert (opts ...QueryOption) string`
- `func (*Migrations) Select (opts ...QueryOption) string`
- `func (*Migrations) SetFilename (val string)`
- `func (*Migrations) SetProject (val string)`
- `func (*Migrations) SetStatementIndex (val int64)`
- `func (*Migrations) SetStatus (val string)`
- `func (*Migrations) Update (opts ...QueryOption) string`
- `func (*Pipeline) GetAliases () map[string]string`
- `func (*Pipeline) GetJobs () map[string]*Job`
- `func (*Pipeline) GetKeys () []string`
- `func (*Pipeline) HasJobs () bool`
- `func (*Pipeline) JobNames () []string`
- `func (*Pipeline) UnmarshalYAML (node *yaml.Node) error`
- `func (*Pipeline) Walk (fn PipelineWalkFunc) error`
- `func (*PipelineWhen) IsEmpty () bool`
- `func (*QueryConfig) Apply (opts ...QueryOption) *QueryConfig`
- `func (*QueryConfig) WithColumns (cols []string) QueryOption`
- `func (*QueryConfig) WithLimit (start,offset int) QueryOption`
- `func (*QueryConfig) WithOrderBy (clause string) QueryOption`
- `func (*QueryConfig) WithStatement (stmt string) QueryOption`
- `func (*QueryConfig) WithTable (name string) QueryOption`
- `func (*QueryConfig) WithWhere (clause string) QueryOption`
- `func (*Run) Delete (opts ...QueryOption) string`
- `func (*Run) GetBranch () string`
- `func (*Run) GetCommitSha () string`
- `func (*Run) GetCreatedAt () *time.Time`
- `func (*Run) GetDir () string`
- `func (*Run) GetDurationMs () int64`
- `func (*Run) GetError () string`
- `func (*Run) GetFile () string`
- `func (*Run) GetFinishedAt () *time.Time`
- `func (*Run) GetID () string`
- `func (*Run) GetInvocationID () string`
- `func (*Run) GetKind () string`
- `func (*Run) GetModule () string`
- `func (*Run) GetName () string`
- `func (*Run) GetParentID () string`
- `func (*Run) GetPipeline () string`
- `func (*Run) GetRepository () string`
- `func (*Run) GetResult () string`
- `func (*Run) GetStartedAt () *time.Time`
- `func (*Run) GetSteps () int64`
- `func (*Run) Insert (opts ...QueryOption) string`
- `func (*Run) Select (opts ...QueryOption) string`
- `func (*Run) SetBranch (val string)`
- `func (*Run) SetCommitSha (val string)`
- `func (*Run) SetCreatedAt (stamp time.Time)`
- `func (*Run) SetDir (val string)`
- `func (*Run) SetDurationMs (val int64)`
- `func (*Run) SetError (val string)`
- `func (*Run) SetFile (val string)`
- `func (*Run) SetFinishedAt (stamp time.Time)`
- `func (*Run) SetID (val string)`
- `func (*Run) SetInvocationID (val string)`
- `func (*Run) SetKind (val string)`
- `func (*Run) SetModule (val string)`
- `func (*Run) SetName (val string)`
- `func (*Run) SetParentID (val string)`
- `func (*Run) SetPipeline (val string)`
- `func (*Run) SetRepository (val string)`
- `func (*Run) SetResult (val string)`
- `func (*Run) SetStartedAt (stamp time.Time)`
- `func (*Run) SetSteps (val int64)`
- `func (*Run) Update (opts ...QueryOption) string`
- `func (*Step) Commands () []string`
- `func (*Step) Dispatch (d StepDispatcher) error`
- `func (*Step) DisplayLabel () string`
- `func (*Step) ExecString () string`
- `func (*Step) IsDeferred () bool`
- `func (*Step) Label (showPrefix bool) *Label`
- `func (*Step) String () string`
- `func (*Step) Tasks () []string`
- `func (*Step) UnmarshalYAML (node *yaml.Node) error`
- `func (Conditionals) IsEmpty () bool`
- `func (Conditionals) String () string`
- `func (Iterators) IsEmpty () bool`

### `NewLabel`

`NewLabel` returns a label with the given text and type.

```go
func NewLabel(text, labelType string) *Label
```

### `NewResolvedTask`

`NewResolvedTask` creates a ResolvedTask with all required fields.

```go
func NewResolvedTask(pipeline *Pipeline, job *Job, name string) *ResolvedTask
```

### `WithColumns`

`WithColumns` will set the columns to use for the query.

```go
func WithColumns(cols []string) QueryOption
```

### `WithLimit`

`WithLimit` will set the limit clause parameters for the query.

```go
func WithLimit(start, offset int) QueryOption
```

### `WithOrderBy`

`WithOrderBy` will set the order by clause for the query.

```go
func WithOrderBy(clause string) QueryOption
```

### `WithStatement`

`WithStatement` will change the statement for the query.

```go
func WithStatement(stmt string) QueryOption
```

### `WithTable`

`WithTable` will set the table name for the query.

```go
func WithTable(name string) QueryOption
```

### `WithWhere`

`WithWhere` will set the where condition for the query.

```go
func WithWhere(clause string) QueryOption
```

### `Delete`

`Delete` starts building a DELETE query.

```go
func (*Command) Delete(opts ...QueryOption) string
```

### `GetCommand`

`GetCommand` will return the value of Command.

```go
func (*Command) GetCommand() string
```

### `GetCreatedAt`

`GetCreatedAt` will return the value of CreatedAt.

```go
func (*Command) GetCreatedAt() *time.Time
```

### `GetDir`

`GetDir` will return the value of Dir.

```go
func (*Command) GetDir() string
```

### `GetDurationMs`

`GetDurationMs` will return the value of DurationMs.

```go
func (*Command) GetDurationMs() int64
```

### `GetError`

`GetError` will return the value of Error.

```go
func (*Command) GetError() string
```

### `GetExitCode`

`GetExitCode` will return the value of ExitCode.

```go
func (*Command) GetExitCode() int64
```

### `GetID`

`GetID` will return the value of ID.

```go
func (*Command) GetID() string
```

### `GetInvocationID`

`GetInvocationID` will return the value of InvocationID.

```go
func (*Command) GetInvocationID() string
```

### `GetKind`

`GetKind` will return the value of Kind.

```go
func (*Command) GetKind() string
```

### `GetNormalized`

`GetNormalized` will return the value of Normalized.

```go
func (*Command) GetNormalized() string
```

### `GetParentID`

`GetParentID` will return the value of ParentID.

```go
func (*Command) GetParentID() string
```

### `GetStartedAt`

`GetStartedAt` will return the value of StartedAt.

```go
func (*Command) GetStartedAt() *time.Time
```

### `GetTool`

`GetTool` will return the value of Tool.

```go
func (*Command) GetTool() string
```

### `Insert`

`Insert` starts building an INSERT INTO query.

```go
func (*Command) Insert(opts ...QueryOption) string
```

### `Select`

`Select` starts building a SELECT query.

```go
func (*Command) Select(opts ...QueryOption) string
```

### `SetCommand`

`SetCommand` sets Command to the provided value.

```go
func (*Command) SetCommand(val string)
```

### `SetCreatedAt`

`SetCreatedAt` sets CreatedAt to the provided value.

```go
func (*Command) SetCreatedAt(stamp time.Time)
```

### `SetDir`

`SetDir` sets Dir to the provided value.

```go
func (*Command) SetDir(val string)
```

### `SetDurationMs`

`SetDurationMs` sets DurationMs to the provided value.

```go
func (*Command) SetDurationMs(val int64)
```

### `SetError`

`SetError` sets Error to the provided value.

```go
func (*Command) SetError(val string)
```

### `SetExitCode`

`SetExitCode` sets ExitCode to the provided value.

```go
func (*Command) SetExitCode(val int64)
```

### `SetID`

`SetID` sets ID to the provided value.

```go
func (*Command) SetID(val string)
```

### `SetInvocationID`

`SetInvocationID` sets InvocationID to the provided value.

```go
func (*Command) SetInvocationID(val string)
```

### `SetKind`

`SetKind` sets Kind to the provided value.

```go
func (*Command) SetKind(val string)
```

### `SetNormalized`

`SetNormalized` sets Normalized to the provided value.

```go
func (*Command) SetNormalized(val string)
```

### `SetParentID`

`SetParentID` sets ParentID to the provided value.

```go
func (*Command) SetParentID(val string)
```

### `SetStartedAt`

`SetStartedAt` sets StartedAt to the provided value.

```go
func (*Command) SetStartedAt(stamp time.Time)
```

### `SetTool`

`SetTool` sets Tool to the provided value.

```go
func (*Command) SetTool(val string)
```

### `Update`

`Update` starts building a UPDATE query.

```go
func (*Command) Update(opts ...QueryOption) string
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for Conditionals,
supporting single string ("enabled == true") or list of strings.

```go
func (*Conditionals) UnmarshalYAML(node *yaml.Node) error
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for `depends_on`,
taking a string value, or a slice of strings.

```go
func (*Dependencies) UnmarshalYAML(node *yaml.Node) error
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for IncludeDecl to support string or []string.

```go
func (*IncludeDecl) UnmarshalYAML(node *yaml.Node) error
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for Iterators,
supporting single string ("item in items") or list of strings.

```go
func (*Iterators) UnmarshalYAML(node *yaml.Node) error
```

### `Children`

`Children` returns job steps for execution.

```go
func (*Job) Children() []*Step
```

### `GetAliases`

`GetAliases` returns the job aliases or nil.

```go
func (*Job) GetAliases() []string
```

### `IsRootLevel`

`IsRootLevel` returns true if the job is a root-level job (no ':' in name).

```go
func (*Job) IsRootLevel() bool
```

### `ShouldShow`

`ShouldShow` returns true if the tree output displays the job.
The tree shows root jobs by default and hides nested jobs unless Show is true.

```go
func (*Job) ShouldShow() bool
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for Job to trim whitespace and handle Decl.
It also supports parsing a job from a simple string (e.g., "up: docker compose up").

```go
func (*Job) UnmarshalYAML(node *yaml.Node) error
```

### `Walk`

`Walk` iterates over all steps (children) of the job.

```go
func (*Job) Walk(fn JobWalkFunc) error
```

### `ForDisplay`

`ForDisplay` returns String with the colour applied and the status appended.

```go
func (*Label) ForDisplay() string
```

### `String`

`String` returns the label text, with the type in front when ShowPrefix holds.

```go
func (*Label) String() string
```

### `WithColor`

`WithColor` sets a color function to apply to the label.

```go
func (*Label) WithColor(colorFn func(string) string) *Label
```

### `WithPrefix`

`WithPrefix` sets whether to show the type prefix in output.

```go
func (*Label) WithPrefix(show bool) *Label
```

### `WithStatus`

`WithStatus` sets the status indicator to display.

```go
func (*Label) WithStatus(status string) *Label
```

### `Delete`

`Delete` starts building a DELETE query.

```go
func (*Migrations) Delete(opts ...QueryOption) string
```

### `GetFilename`

`GetFilename` will return the value of Filename.

```go
func (*Migrations) GetFilename() string
```

### `GetProject`

`GetProject` will return the value of Project.

```go
func (*Migrations) GetProject() string
```

### `GetStatementIndex`

`GetStatementIndex` will return the value of StatementIndex.

```go
func (*Migrations) GetStatementIndex() int64
```

### `GetStatus`

`GetStatus` will return the value of Status.

```go
func (*Migrations) GetStatus() string
```

### `Insert`

`Insert` starts building an INSERT INTO query.

```go
func (*Migrations) Insert(opts ...QueryOption) string
```

### `Select`

`Select` starts building a SELECT query.

```go
func (*Migrations) Select(opts ...QueryOption) string
```

### `SetFilename`

`SetFilename` sets Filename to the provided value.

```go
func (*Migrations) SetFilename(val string)
```

### `SetProject`

`SetProject` sets Project to the provided value.

```go
func (*Migrations) SetProject(val string)
```

### `SetStatementIndex`

`SetStatementIndex` sets StatementIndex to the provided value.

```go
func (*Migrations) SetStatementIndex(val int64)
```

### `SetStatus`

`SetStatus` sets Status to the provided value.

```go
func (*Migrations) SetStatus(val string)
```

### `Update`

`Update` starts building a UPDATE query.

```go
func (*Migrations) Update(opts ...QueryOption) string
```

### `GetAliases`

`GetAliases` returns alias => job name, explicit job aliases ahead of the generated ones.

```go
func (*Pipeline) GetAliases() map[string]string
```

### `GetJobs`

`GetJobs` returns Jobs, or Tasks when Jobs is empty.

```go
func (*Pipeline) GetJobs() map[string]*Job
```

### `GetKeys`

`GetKeys` returns the sorted job names, "default" first, prefixed with "ID:" for a pipeline with an ID.

```go
func (*Pipeline) GetKeys() []string
```

### `HasJobs`

`HasJobs` returns true if the pipeline has any jobs or tasks defined.

```go
func (*Pipeline) HasJobs() bool
```

### `JobNames`

`JobNames` returns the names of all jobs (or tasks) in the pipeline.

```go
func (*Pipeline) JobNames() []string
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for Pipeline to handle Decl.

```go
func (*Pipeline) UnmarshalYAML(node *yaml.Node) error
```

### `Walk`

`Walk` returns the first error fn returns, visiting "default" and then the rest sorted.
Walk returns nil once fn has visited every job.

```go
func (*Pipeline) Walk(fn PipelineWalkFunc) error
```

### `IsEmpty`

`IsEmpty` reports whether the block lists no files, leaving the pipeline or job active everywhere.

```go
func (*PipelineWhen) IsEmpty() bool
```

### `Apply`

`Apply` will use passed query options to populate the query.

```go
func (*QueryConfig) Apply(opts ...QueryOption) *QueryConfig
```

### `WithColumns`

`WithColumns` will set the columns to use for the query.

```go
func (*QueryConfig) WithColumns(cols []string) QueryOption
```

### `WithLimit`

`WithLimit` will set the limit clause parameters for the query.

```go
func (*QueryConfig) WithLimit(start, offset int) QueryOption
```

### `WithOrderBy`

`WithOrderBy` will set the order by clause for the query.

```go
func (*QueryConfig) WithOrderBy(clause string) QueryOption
```

### `WithStatement`

`WithStatement` will change the statement for the query.

```go
func (*QueryConfig) WithStatement(stmt string) QueryOption
```

### `WithTable`

`WithTable` will set the table name for the query.

```go
func (*QueryConfig) WithTable(name string) QueryOption
```

### `WithWhere`

`WithWhere` will set the where condition for the query.

```go
func (*QueryConfig) WithWhere(clause string) QueryOption
```

### `Delete`

`Delete` starts building a DELETE query.

```go
func (*Run) Delete(opts ...QueryOption) string
```

### `GetBranch`

`GetBranch` will return the value of Branch.

```go
func (*Run) GetBranch() string
```

### `GetCommitSha`

`GetCommitSha` will return the value of CommitSha.

```go
func (*Run) GetCommitSha() string
```

### `GetCreatedAt`

`GetCreatedAt` will return the value of CreatedAt.

```go
func (*Run) GetCreatedAt() *time.Time
```

### `GetDir`

`GetDir` will return the value of Dir.

```go
func (*Run) GetDir() string
```

### `GetDurationMs`

`GetDurationMs` will return the value of DurationMs.

```go
func (*Run) GetDurationMs() int64
```

### `GetError`

`GetError` will return the value of Error.

```go
func (*Run) GetError() string
```

### `GetFile`

`GetFile` will return the value of File.

```go
func (*Run) GetFile() string
```

### `GetFinishedAt`

`GetFinishedAt` will return the value of FinishedAt.

```go
func (*Run) GetFinishedAt() *time.Time
```

### `GetID`

`GetID` will return the value of ID.

```go
func (*Run) GetID() string
```

### `GetInvocationID`

`GetInvocationID` will return the value of InvocationID.

```go
func (*Run) GetInvocationID() string
```

### `GetKind`

`GetKind` will return the value of Kind.

```go
func (*Run) GetKind() string
```

### `GetModule`

`GetModule` will return the value of Module.

```go
func (*Run) GetModule() string
```

### `GetName`

`GetName` will return the value of Name.

```go
func (*Run) GetName() string
```

### `GetParentID`

`GetParentID` will return the value of ParentID.

```go
func (*Run) GetParentID() string
```

### `GetPipeline`

`GetPipeline` will return the value of Pipeline.

```go
func (*Run) GetPipeline() string
```

### `GetRepository`

`GetRepository` will return the value of Repository.

```go
func (*Run) GetRepository() string
```

### `GetResult`

`GetResult` will return the value of Result.

```go
func (*Run) GetResult() string
```

### `GetStartedAt`

`GetStartedAt` will return the value of StartedAt.

```go
func (*Run) GetStartedAt() *time.Time
```

### `GetSteps`

`GetSteps` will return the value of Steps.

```go
func (*Run) GetSteps() int64
```

### `Insert`

`Insert` starts building an INSERT INTO query.

```go
func (*Run) Insert(opts ...QueryOption) string
```

### `Select`

`Select` starts building a SELECT query.

```go
func (*Run) Select(opts ...QueryOption) string
```

### `SetBranch`

`SetBranch` sets Branch to the provided value.

```go
func (*Run) SetBranch(val string)
```

### `SetCommitSha`

`SetCommitSha` sets CommitSha to the provided value.

```go
func (*Run) SetCommitSha(val string)
```

### `SetCreatedAt`

`SetCreatedAt` sets CreatedAt to the provided value.

```go
func (*Run) SetCreatedAt(stamp time.Time)
```

### `SetDir`

`SetDir` sets Dir to the provided value.

```go
func (*Run) SetDir(val string)
```

### `SetDurationMs`

`SetDurationMs` sets DurationMs to the provided value.

```go
func (*Run) SetDurationMs(val int64)
```

### `SetError`

`SetError` sets Error to the provided value.

```go
func (*Run) SetError(val string)
```

### `SetFile`

`SetFile` sets File to the provided value.

```go
func (*Run) SetFile(val string)
```

### `SetFinishedAt`

`SetFinishedAt` sets FinishedAt to the provided value.

```go
func (*Run) SetFinishedAt(stamp time.Time)
```

### `SetID`

`SetID` sets ID to the provided value.

```go
func (*Run) SetID(val string)
```

### `SetInvocationID`

`SetInvocationID` sets InvocationID to the provided value.

```go
func (*Run) SetInvocationID(val string)
```

### `SetKind`

`SetKind` sets Kind to the provided value.

```go
func (*Run) SetKind(val string)
```

### `SetModule`

`SetModule` sets Module to the provided value.

```go
func (*Run) SetModule(val string)
```

### `SetName`

`SetName` sets Name to the provided value.

```go
func (*Run) SetName(val string)
```

### `SetParentID`

`SetParentID` sets ParentID to the provided value.

```go
func (*Run) SetParentID(val string)
```

### `SetPipeline`

`SetPipeline` sets Pipeline to the provided value.

```go
func (*Run) SetPipeline(val string)
```

### `SetRepository`

`SetRepository` sets Repository to the provided value.

```go
func (*Run) SetRepository(val string)
```

### `SetResult`

`SetResult` sets Result to the provided value.

```go
func (*Run) SetResult(val string)
```

### `SetStartedAt`

`SetStartedAt` sets StartedAt to the provided value.

```go
func (*Run) SetStartedAt(stamp time.Time)
```

### `SetSteps`

`SetSteps` sets Steps to the provided value.

```go
func (*Run) SetSteps(val int64)
```

### `Update`

`Update` starts building a UPDATE query.

```go
func (*Run) Update(opts ...QueryOption) string
```

### `Commands`

`Commands` returns all executable commands from this step as a slice.
For steps with a single command (Run/Cmd), returns a slice with one element.
For steps with a command list (Cmds), returns the full slice.
An exec: step returns its argv rendered as one display string.
Returns an empty slice for Task steps.

```go
func (*Step) Commands() []string
```

### `Dispatch`

`Dispatch` returns d.Task for a task: step.
Dispatch returns the first error of d.Command, called once per command.
Dispatch returns nil once every command has run.

```go
func (*Step) Dispatch(d StepDispatcher) error
```

### `DisplayLabel`

`DisplayLabel` returns a display label for the step, always showing prefixes.
Execution uses it to show which kind of operation runs.

```go
func (*Step) DisplayLabel() string
```

### `ExecString`

`ExecString` returns the exec: argv as one display string.

```go
func (*Step) ExecString() string
```

### `IsDeferred`

`IsDeferred` returns true if the step sets deferred.

```go
func (*Step) IsDeferred() bool
```

### `Label`

`Label` returns a structured label for the step with information about how to display it.
showPrefix controls whether to include the type prefix (e.g., "run:", "task:").
HidePrefix hides the prefix in every case (for simple shorthand tasks).

```go
func (*Step) Label(showPrefix bool) *Label
```

### `String`

`String` returns a string representation of the step.

```go
func (*Step) String() string
```

### `Tasks`

`Tasks` returns the job names of the task: field, none for a command step.

```go
func (*Step) Tasks() []string
```

### `UnmarshalYAML`

`UnmarshalYAML` implements custom unmarshalling for Step to accept the shorthand formats and decode Decl.

```go
func (*Step) UnmarshalYAML(node *yaml.Node) error
```

### `IsEmpty`

`IsEmpty` returns true if there are no conditions.

```go
func (Conditionals) IsEmpty() bool
```

### `String`

`String` returns a display representation of the conditions.
Single condition returns the expression as-is.
More than one condition returns each in parentheses, joined with " && ".

```go
func (Conditionals) String() string
```

### `IsEmpty`

`IsEmpty` returns true if there are no iterators.

```go
func (Iterators) IsEmpty() bool
```
