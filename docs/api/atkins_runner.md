# Package ./runner

```go
import (
	"github.com/titpetric/atkins/runner"
}
```

## Types

<details>
<summary><code>type ContextVariables</code></summary>

```go
// ContextVariables provides thread-safe variable storage with Promise-based lazy evaluation.
type ContextVariables struct {
	promises        map[string]*VarPromise
	resolver        func(string) (string, error)
	loopResolvedEnv map[string]bool
	mu              sync.Mutex
}
```

</details>

<details>
<summary><code>type Env</code></summary>

```go
// Env is a map of environment variables.
type Env map[string]string
```

</details>

<details>
<summary><code>type Environment</code></summary>

```go
// Environment represents the discovered project environment.
type Environment struct {
	Root string // Project root directory
}
```

</details>

<details>
<summary><code>type ExecError</code></summary>

```go
// ExecError represents an error from command execution.
type ExecError struct {
	Message      string
	Output       string
	LastExitCode int
}
```

</details>

<details>
<summary><code>type ExecutionContext</code></summary>

```go
// ExecutionContext holds runtime state during pipeline Exec.
type ExecutionContext struct {
	Context context.Context

	Env     Env
	Results map[string]any
	Verbose bool
	Dir     string

	Variables model.VariableStorage

	Pipeline     *model.Pipeline
	AllPipelines []*model.Pipeline // All loaded pipelines for cross-pipeline task references
	Job          *model.Job
	Step         *model.Step

	Depth       int // Nesting depth for indentation
	StepsCount  int // Total number of steps executed
	StepsPassed int // Number of steps that passed

	CurrentJob  *treeview.TreeNode
	CurrentStep *treeview.Node

	Display     *treeview.Display
	Builder     *treeview.Builder
	JobNodes    map[string]*treeview.TreeNode // Map of job names to their tree nodes
	EventLogger *eventlog.Logger

	// Sequential step counter for this job (incremented for each step execution)
	StepSequence int
	stepSeqMu    sync.Mutex

	// jobTracker tracks which jobs have finished execution (for dependency resolution).
	// Shared across copies so the mutex protects the map consistently.
	jobTracker *jobTracker

	// Progress receives job lifecycle events (optional).
	Progress ProgressObserver

	// Parents is the ancestor job chain for nested task invocations.
	Parents []string
}
```

</details>

<details>
<summary><code>type Executor</code></summary>

```go
// Executor runs pipeline jobs and steps.
type Executor struct {
	opts *Options
}
```

</details>

<details>
<summary><code>type FuzzyMatchError</code></summary>

```go
// FuzzyMatchError is returned when multiple fuzzy matches are found.
type FuzzyMatchError struct {
	Matches []*model.ResolvedTask
}
```

</details>

<details>
<summary><code>type IterationContext</code></summary>

```go
// IterationContext holds the variables for a single iteration of a for loop.
type IterationContext struct {
	Variables model.VariableStorage
}
```

</details>

<details>
<summary><code>type JobProgressEvent</code></summary>

```go
// JobProgressEvent represents a job lifecycle event.
type JobProgressEvent struct {
	JobName   string
	Parents   []string // ancestor chain for nested task invocations, e.g. ["default", "fmt"] when "lint" runs inside default > fmt
	Status    JobProgressStatus
	StartedAt time.Time
	Duration  time.Duration // set for terminal states
	Err       error         // set for failed
}
```

</details>

<details>
<summary><code>type JobProgressStatus</code></summary>

```go
// JobProgressStatus represents the status of a job in progress.
type JobProgressStatus string
```

</details>

<details>
<summary><code>type LineCapturingWriter</code></summary>

```go
// LineCapturingWriter captures all output written to it.
type LineCapturingWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}
```

</details>

<details>
<summary><code>type LintError</code></summary>

```go
// LintError represents a linting error.
type LintError struct {
	Job    string
	Issue  string
	Detail string
}
```

</details>

<details>
<summary><code>type Linter</code></summary>

```go
// Linter validates a pipeline for correctness.
type Linter struct {
	pipeline     *model.Pipeline
	allPipelines []*model.Pipeline // All pipelines for cross-pipeline validation
	errors       []LintError
}
```

</details>

<details>
<summary><code>type Options</code></summary>

```go
// Options provides configuration for the executor.
type Options struct {
	DefaultTimeout time.Duration
}
```

</details>

<details>
<summary><code>type OutputItem</code></summary>

```go
// OutputItem represents a single command in the list output.
type OutputItem struct {
	ID   string `json:"id" yaml:"id"`
	Desc string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Cmd  string `json:"cmd" yaml:"cmd"`
}
```

</details>

<details>
<summary><code>type OutputSection</code></summary>

```go
// OutputSection represents a pipeline section in the list output.
type OutputSection struct {
	Desc string       `json:"desc" yaml:"desc"`
	Cmds []OutputItem `json:"cmds" yaml:"cmds"`
}
```

</details>

<details>
<summary><code>type Pipeline</code></summary>

```go
// Pipeline holds pipeline execution logic.
type Pipeline struct {
	opts PipelineOptions
	data *model.Pipeline
}
```

</details>

<details>
<summary><code>type PipelineOptions</code></summary>

```go
// PipelineOptions contains options for running a pipeline.
type PipelineOptions struct {
	Jobs         []string // Jobs to run (in order)
	LogFile      string
	PipelineFile string
	Debug        bool
	FinalOnly    bool
	Silent       bool
	JSON         bool
	YAML         bool
	AllPipelines []*model.Pipeline // All loaded pipelines for cross-pipeline task references
	Progress     ProgressObserver  // Optional observer for job progress events

	// Transcript receives the finished tree, the same rendering the
	// terminal is left with. It is how a caller records the run
	// somewhere else — a CI job log — without changing what a person
	// watching it sees.
	Transcript io.Writer
}
```

</details>

<details>
<summary><code>type ProgressObserver</code></summary>

```go
// ProgressObserver receives job progress events.
type ProgressObserver interface {
	OnJobProgress(JobProgressEvent)
}
```

</details>

<details>
<summary><code>type ProgressObserverFunc</code></summary>

```go
// ProgressObserverFunc is a function adapter for ProgressObserver.
type ProgressObserverFunc func(JobProgressEvent)
```

</details>

<details>
<summary><code>type Resolver</code></summary>

```go
// Resolver resolves vars and env.vars together using a unified dependency
// graph. This handles cross-dependencies where vars use $(echo $ENV_VAR)
// and env uses ${{ var_name }}.
type Resolver struct {
	vars    map[string]any
	envVars map[string]any

	baseVars map[string]any
	baseEnv  map[string]string

	workCtx *ExecutionContext
}
```

</details>

<details>
<summary><code>type Skill</code></summary>

```go
// Skill is one skill file found on disk.
// A skill is a pipeline file whose name is the job namespace: go.yml
// contributes go:build, go:test and the rest. Beside it a skill may
// carry a markdown companion of the same base name, go.md, which is a
// usage guide rather than a job list; `atkins --help` prints it under
// the skill.
type Skill struct {
	// Pipeline is the parsed skill. Its ID is the file name without the
	// extension, and prefixes every job the skill declares.
	Pipeline *model.Pipeline

	// Path is the absolute path of the skill file.
	Path string

	// DocPath is the markdown companion beside the skill file, empty
	// when the skill carries none.
	DocPath string

	// Doc is the contents of DocPath, with surrounding blank lines
	// trimmed.
	Doc string

	// Dir is the directory the skill runs in: the folder that satisfied
	// its when: block, or the workspace root when it has none. It is
	// only resolved for an active skill.
	Dir string

	// Active reports whether the skill's when: block is satisfied from
	// the directory the scan started in. Only an active skill
	// contributes its jobs to a run.
	Active bool
}
```

</details>

<details>
<summary><code>type SkillsLoader</code></summary>

```go
// SkillsLoader discovers and loads skill pipelines from .atkins/skills/ directories.
// It evaluates `when:` conditions to determine which skills are enabled and sets
// the appropriate working directory for each skill based on the rules.
type SkillsLoader struct {
	// SkillsDirs are the directories to search for skill files, in priority order.
	// First directory takes precedence for skills with the same ID.
	SkillsDirs []string

	// StartDir is the directory from which to start searching for when: files.
	// This is typically the user's working directory.
	StartDir string

	// WorkspaceDir is the folder containing .atkins/ (used for skills without when:).
	WorkspaceDir string
}
```

</details>

<details>
<summary><code>type TaskResolver</code></summary>

```go
// TaskResolver resolves task references, handling cross-pipeline : prefix syntax.
type TaskResolver struct {
	pipelines []*model.Pipeline
}
```

</details>

<details>
<summary><code>type VarPromise</code></summary>

```go
// VarPromise represents a lazy variable that is evaluated on first access.
type VarPromise struct {
	value any
	raw   any
	err   error
	state promiseState
	mu    sync.Mutex
}
```

</details>

<details>
<summary><code>type VendorResult</code></summary>

```go
// VendorResult reports what a vendoring run found, matched and wrote.
type VendorResult struct {
	// SourceDir is the skills directory that was read.
	SourceDir string

	// TargetDir is the repository directory that was written to.
	TargetDir string

	// Found are all skills available in SourceDir.
	Found []*VendorSkill

	// Used are the skills this repository has a use for.
	Used []*VendorSkill

	// Installed are the IDs written to TargetDir. It stays empty on a
	// dry run; Pending says what a write would have done.
	Installed []string
}
```

</details>

<details>
<summary><code>type VendorSkill</code></summary>

```go
// VendorSkill is a candidate skill file, with the reason it was picked
// for vendoring once it has been matched against a repository.
type VendorSkill struct {
	// ID is the skill namespace, taken from the file name.
	ID string

	// Path is the absolute path of the source skill file.
	Path string

	// DocPath is the markdown companion beside Path, empty when the
	// skill carries none. It is installed alongside the skill file.
	DocPath string

	// Pipeline is the parsed skill.
	Pipeline *model.Pipeline

	// Reason records why the skill is used here, e.g. the path that
	// satisfied its when: block, or the pipeline that references it.
	Reason string

	// Status is what installing this skill would do, set for used skills.
	Status VendorStatus

	// Added and Removed are the line counts installing this skill would
	// add to and remove from the vendored copy.
	Added   int
	Removed int

	// Diff is the unified diff from the vendored copy to the source,
	// empty when there is nothing to change.
	Diff string
}
```

</details>

<details>
<summary><code>type VendorStatus</code></summary>

```go
// VendorStatus is what installing a skill would do to the repository.
type VendorStatus string
```

</details>

<details>
<summary><code>type Vendorer</code></summary>

```go
// Vendorer copies the skills a repository uses out of a shared skills
// directory and into the repository itself, so a clone or a CI agent
// gets the same jobs without a personal $HOME/.atkins/skills.
type Vendorer struct {
	// SourceDir is the skills directory to vendor from, typically
	// $HOME/.atkins/skills.
	SourceDir string

	// Root is the repository root, the folder holding .git.
	Root string

	// TargetDir is where skills are written, Root/.atkins/skills.
	TargetDir string

	// SkipDirs are directory names not descended into.
	SkipDirs []string
}
```

</details>

## Consts

<details>
<summary><code>const JobProgressRunning, JobProgressPassed, JobProgressFailed, JobProgressSkipped</code></summary>

```go
// Job progress status constants.
const (
	JobProgressRunning JobProgressStatus = "running"
	JobProgressPassed  JobProgressStatus = "passed"
	JobProgressFailed  JobProgressStatus = "failed"
	JobProgressSkipped JobProgressStatus = "skipped"
)
```

</details>

<details>
<summary><code>const VendorNew, VendorCurrent, VendorChanged</code></summary>

```go
const (
	// VendorNew means the repository doesn't carry the skill yet.
	VendorNew VendorStatus = "new"

	// VendorCurrent means an identical copy is already vendored.
	VendorCurrent VendorStatus = "up to date"

	// VendorChanged means the vendored copy differs from the source and
	// would be overwritten.
	VendorChanged VendorStatus = "changed"
)
```

</details>

## Vars

<details>
<summary><code>var ConfigNames</code></summary>

```go
// ConfigNames are the default config file names to search for, in order of preference.
var ConfigNames = []string{".atkins.yml", ".atkins.yaml", "atkins.yml", "atkins.yaml"}
```

</details>

<details>
<summary><code>var ErrJobSkipped</code></summary>

```go
// ErrJobSkipped is returned when a job's if condition evaluates to false.
var ErrJobSkipped = errors.New("job skipped")
```

</details>

## Function symbols

- `func DefaultOptions () *Options`
- `func DiscoverConfig (startDir string) (string, error)`
- `func DiscoverConfigFromCwd () (string, error)`
- `func DiscoverEnvironment (startDir string) (*Environment, error)`
- `func DiscoverEnvironmentFromCwd () (*Environment, error)`
- `func EvaluateIf (ctx *ExecutionContext) (bool, error)`
- `func EvaluateJobIf (ctx *ExecutionContext) (bool, error)`
- `func ExpandFor (ctx *ExecutionContext, executeCommand func(string) (string, error)) ([]IterationContext, error)`
- `func FindRepositoryRoot (startDir string) (string, error)`
- `func GetDependencies (dependsOn any) []string`
- `func InterpolateCommand (cmd string, ctx *ExecutionContext) (string, error)`
- `func InterpolateMap (ctx *ExecutionContext, m map[string]any) error`
- `func InterpolateString (s string, ctx *ExecutionContext) (string, error)`
- `func IsEchoCommand (cmd string) bool`
- `func ListPipelines (pipelines []*model.Pipeline) string`
- `func ListPipelinesJSON (pipelines []*model.Pipeline) error`
- `func ListPipelinesYAML (pipelines []*model.Pipeline) error`
- `func LoadPipeline (filePath string) ([]*model.Pipeline, error)`
- `func LoadPipelineFromReader (r io.Reader) ([]*model.Pipeline, error)`
- `func MatchWhenPattern (dir,pattern string) (string, bool)`
- `func MergeSkillVariables (ctx *ExecutionContext, decl *model.Decl) error`
- `func MergeVariables (ctx *ExecutionContext, decl *model.Decl) error`
- `func NewContextVariables (values map[string]any) *ContextVariables`
- `func NewContextVariablesWithResolver (pending map[string]any, resolver func(string) (string, error)) *ContextVariables`
- `func NewEnv (environ []string) Env`
- `func NewExecError (result psexec.Result) ExecError`
- `func NewExecutor () *Executor`
- `func NewExecutorWithOptions (opts *Options) *Executor`
- `func NewLineCapturingWriter () *LineCapturingWriter`
- `func NewLinter (pipeline *model.Pipeline) *Linter`
- `func NewLinterWithPipelines (pipeline *model.Pipeline, allPipelines []*model.Pipeline) *Linter`
- `func NewPipeline (data *model.Pipeline, opts PipelineOptions) *Pipeline`
- `func NewSkillResolver (pipeline *model.Pipeline) *TaskResolver`
- `func NewSkillsLoader (workspaceDir,startDir string) *SkillsLoader`
- `func NewTaskResolver (pipelines []*model.Pipeline) *TaskResolver`
- `func NewVendorer (sourceDir,root string) *Vendorer`
- `func ProcessDecl (ctx *ExecutionContext, decl *model.Decl) (map[string]any, error)`
- `func ResolveJobDependencies (jobs map[string]*model.Job, startingJob string) ([]string, error)`
- `func RunPipeline (ctx context.Context, pipeline *model.Pipeline, opts PipelineOptions) error`
- `func Sanitize (in string) ([]string, error)`
- `func StripANSI (in string) string`
- `func ValidateJobRequirements (ctx *ExecutionContext, job *model.Job) error`
- `func VisualLength (s string) int`
- `func (*ContextVariables) Clone () model.VariableStorage`
- `func (*ContextVariables) Get (key string) any`
- `func (*ContextVariables) GetWithError (key string) (any, error)`
- `func (*ContextVariables) IsResolved (key string) bool`
- `func (*ContextVariables) ResolveAll () error`
- `func (*ContextVariables) Set (key string, value any)`
- `func (*ContextVariables) SetResolver (resolver func(string) (string, error))`
- `func (*ContextVariables) Walk (fn func(key string, value any))`
- `func (*ExecutionContext) Copy () *ExecutionContext`
- `func (*ExecutionContext) EmitProgress (ev JobProgressEvent)`
- `func (*ExecutionContext) IsJobCompleted (jobName string) bool`
- `func (*ExecutionContext) MarkJobCompleted (jobName string)`
- `func (*ExecutionContext) NextStepIndex () int`
- `func (*ExecutionContext) Render ()`
- `func (*ExecutionContext) Resolve (taskName string) (*model.ResolvedTask, error)`
- `func (*ExecutionContext) Resolver () *TaskResolver`
- `func (*ExecutionContext) SkillResolver () *TaskResolver`
- `func (*Executor) ExecuteJob (parentCtx context.Context, execCtx *ExecutionContext) error`
- `func (*FuzzyMatchError) Error () string`
- `func (*LineCapturingWriter) GetLines () []string`
- `func (*LineCapturingWriter) String () string`
- `func (*LineCapturingWriter) Write (p []byte) (int, error)`
- `func (*Linter) Lint () []LintError`
- `func (*Skill) ID () string`
- `func (*SkillsLoader) AddSkillsDir (dir string)`
- `func (*SkillsLoader) FindFile (patterns []string, startDir string) (string, bool)`
- `func (*SkillsLoader) FindFolder (name,startDir string) (string, bool)`
- `func (*SkillsLoader) Load () ([]*model.Pipeline, error)`
- `func (*SkillsLoader) Scan () ([]*Skill, error)`
- `func (*TaskResolver) Resolve (taskName string) (*model.ResolvedTask, error)`
- `func (*TaskResolver) ResolveName (name string, strict bool) (*model.ResolvedTask, error)`
- `func (*TaskResolver) ResolveWithFallback (taskName string, fallback *TaskResolver) (*model.ResolvedTask, error)`
- `func (*VendorResult) Pending () []string`
- `func (*Vendorer) Install (result *VendorResult) error`
- `func (*Vendorer) Plan () (*VendorResult, error)`
- `func (*Vendorer) Run () (*VendorResult, error)`
- `func (Env) Environ () []string`
- `func (Env) Sanitized () Env`
- `func (ExecError) Error () string`
- `func (ExecError) Len () int`
- `func (ProgressObserverFunc) OnJobProgress (ev JobProgressEvent)`

### DefaultOptions

DefaultOptions returns the default executor options.

```go
func DefaultOptions() *Options
```

### DiscoverConfig

DiscoverConfig searches for a config file starting from the given directory,
traversing parent directories until a config file is found or root is reached.
Returns the absolute path to the config file and the directory containing it.

```go
func DiscoverConfig(startDir string) (string, error)
```

### DiscoverConfigFromCwd

DiscoverConfigFromCwd is a convenience wrapper that starts from the current working directory.

```go
func DiscoverConfigFromCwd() (string, error)
```

### DiscoverEnvironment

DiscoverEnvironment scans for marker files starting from startDir,
traversing parent directories until the filesystem root is reached.
Root is set to the highest directory that contains any markers.

```go
func DiscoverEnvironment(startDir string) (*Environment, error)
```

### DiscoverEnvironmentFromCwd

DiscoverEnvironmentFromCwd is a convenience wrapper that starts from the current working directory.

```go
func DiscoverEnvironmentFromCwd() (*Environment, error)
```

### EvaluateIf

EvaluateIf evaluates the If condition using expr-lang.
Returns true if the condition is met, false if no condition or condition is false.
When multiple conditions are provided, all must be true (AND logic).
Returns error only for invalid expressions.

```go
func EvaluateIf(ctx *ExecutionContext) (bool, error)
```

### EvaluateJobIf

EvaluateJobIf evaluates the If condition on a job using expr-lang.
Returns true if the condition is met or no condition is set.
When multiple conditions are provided, all must be true (AND logic).
Returns error only for invalid expressions.

```go
func EvaluateJobIf(ctx *ExecutionContext) (bool, error)
```

### ExpandFor

ExpandFor expands a for loop into multiple iteration contexts.
Supports patterns: "item in items" (items is a variable name),
"(index, item) in items", "(key, value) in items",
or any of the above with bash expansion: "item in $(ls ./bin/*.test)".
When multiple iterators are provided, computes the cartesian product.

```go
func ExpandFor(ctx *ExecutionContext, executeCommand func(string) (string, error)) ([]IterationContext, error)
```

### FindRepositoryRoot

FindRepositoryRoot returns the folder holding .git, starting at
startDir and traversing parent directories.

```go
func FindRepositoryRoot(startDir string) (string, error)
```

### GetDependencies

GetDependencies converts depends_on field (string or []string) to a slice of job names.

```go
func GetDependencies(dependsOn any) []string
```

### InterpolateCommand

InterpolateCommand interpolates a command string.

```go
func InterpolateCommand(cmd string, ctx *ExecutionContext) (string, error)
```

### InterpolateMap

InterpolateMap recursively interpolates all string values in a map.

```go
func InterpolateMap(ctx *ExecutionContext, m map[string]any) error
```

### InterpolateString

InterpolateString replaces ${{ expression }} with values from context.
Supports variable interpolation, dot notation, and expr expressions with ?? and || operators.

```go
func InterpolateString(s string, ctx *ExecutionContext) (string, error)
```

### IsEchoCommand

IsEchoCommand checks if a command is a bare echo command.

```go
func IsEchoCommand(cmd string) bool
```

### ListPipelines

ListPipelines returns pipelines formatted as a string in a flat list format:
Main Pipeline, then Aliases, then Skills.

```go
func ListPipelines(pipelines []*model.Pipeline) string
```

### ListPipelinesJSON

ListPipelinesJSON outputs pipelines in JSON format.

```go
func ListPipelinesJSON(pipelines []*model.Pipeline) error
```

### ListPipelinesYAML

ListPipelinesYAML outputs pipelines in YAML format.

```go
func ListPipelinesYAML(pipelines []*model.Pipeline) error
```

### LoadPipeline

LoadPipeline loads and parses a pipeline from a yaml file.
Returns the number of documents loaded, the parsed pipeline, and any error.

```go
func LoadPipeline(filePath string) ([]*model.Pipeline, error)
```

### LoadPipelineFromReader

LoadPipelineFromReader loads and parses a pipeline from an io.Reader.
Returns the parsed pipeline(s) and any error.

```go
func LoadPipelineFromReader(r io.Reader) ([]*model.Pipeline, error)
```

### MatchWhenPattern

MatchWhenPattern reports whether a `when: files:` pattern resolves to
something inside dir, and what it resolved to.

A pattern carrying a glob meta character is expanded, so a skill can
activate on the contents of a folder rather than its name:
`schema/*.up.sql` matches a folder of migrations, while `schema/`
matches any folder of that name.

```go
func MatchWhenPattern(dir, pattern string) (string, bool)
```

### MergeSkillVariables

MergeSkillVariables merges variables from a skill's Decl into the context.
When depth > 1, variables are already on the stack from a parent pipeline,
so new vars are treated as defaults (existing keys are preserved).
At depth <= 1 it behaves identically to MergeVariables.

```go
func MergeSkillVariables(ctx *ExecutionContext, decl *model.Decl) error
```

### MergeVariables

MergeVariables merges variables from Decl into the execution context.
When both vars and env.vars are present, they are resolved together using
a unified dependency graph so that cross-references work correctly
(e.g., vars using $(echo $ENV_VAR) and env using ${{ var_name }}).

```go
func MergeVariables(ctx *ExecutionContext, decl *model.Decl) error
```

### NewContextVariables

NewContextVariables creates a ContextVariables from a map of evaluated values.

```go
func NewContextVariables(values map[string]any) *ContextVariables
```

### NewContextVariablesWithResolver

NewContextVariablesWithResolver creates a ContextVariables with pending values
that are evaluated lazily on first access via Get().

```go
func NewContextVariablesWithResolver(pending map[string]any, resolver func(string) (string, error)) *ContextVariables
```

### NewEnv

NewEnv builds an Env from KEY=VALUE strings, the form os.Environ and
exec.Cmd use. The caller supplies the slice, so a process
environment, a captured one or a literal all read the same way.

A later entry wins, matching what exec does with a duplicate name.
Entries without a name are skipped.

```go
func NewEnv(environ []string) Env
```

### NewExecError

NewExecError creates an ExecError from a psexec.Result.

```go
func NewExecError(result psexec.Result) ExecError
```

### NewExecutor

NewExecutor creates a new executor with default options.

```go
func NewExecutor() *Executor
```

### NewExecutorWithOptions

NewExecutorWithOptions creates a new executor with custom options.

```go
func NewExecutorWithOptions(opts *Options) *Executor
```

### NewLineCapturingWriter

NewLineCapturingWriter creates a new LineCapturingWriter.

```go
func NewLineCapturingWriter() *LineCapturingWriter
```

### NewLinter

NewLinter creates a new linter.

```go
func NewLinter(pipeline *model.Pipeline) *Linter
```

### NewLinterWithPipelines

NewLinterWithPipelines creates a linter with access to all pipelines for cross-pipeline validation.

```go
func NewLinterWithPipelines(pipeline *model.Pipeline, allPipelines []*model.Pipeline) *Linter
```

### NewPipeline

NewPipeline allocates a new *Pipeline with dependencies.

```go
func NewPipeline(data *model.Pipeline, opts PipelineOptions) *Pipeline
```

### NewSkillResolver

NewSkillResolver will provide a task resolver for a skill pipeline.

```go
func NewSkillResolver(pipeline *model.Pipeline) *TaskResolver
```

### NewSkillsLoader

NewSkillsLoader creates a loader for the given workspace.
workspaceDir is the folder containing .atkins/ (used as Dir for skills without when:).
startDir is where to start searching for when: files (typically user's cwd).

```go
func NewSkillsLoader(workspaceDir, startDir string) *SkillsLoader
```

### NewTaskResolver

NewTaskResolver will provide a task resolver for a set of pipelines.

```go
func NewTaskResolver(pipelines []*model.Pipeline) *TaskResolver
```

### NewVendorer

NewVendorer creates a vendorer writing into root/.atkins/skills.

```go
func NewVendorer(sourceDir, root string) *Vendorer
```

### ProcessDecl

ProcessDecl processes a Decl and returns a map of variables.
It handles:
- Manual vars with interpolation ($(...), ${{ ... }})
- Include files (.yml format)
  Vars take precedence over included files.

```go
func ProcessDecl(ctx *ExecutionContext, decl *model.Decl) (map[string]any, error)
```

### ResolveJobDependencies

ResolveJobDependencies returns jobs in dependency order.
Returns the jobs to run and any resolution errors.

```go
func ResolveJobDependencies(jobs map[string]*model.Job, startingJob string) ([]string, error)
```

### RunPipeline

RunPipeline runs a pipeline with the given options.

```go
func RunPipeline(ctx context.Context, pipeline *model.Pipeline, opts PipelineOptions) error
```

### Sanitize

Sanitize processes raw terminal output and returns clean lines.
It handles:
- Cursor up + clear sequences (\033[nA\033[J) used by treeview
- Carriage returns (\r) by taking content after the last \r
- CRLF normalization
- Preserves ANSI color sequences in output

Returns sanitized lines with colors preserved.

```go
func Sanitize(in string) ([]string, error)
```

### StripANSI

StripANSI removes all ANSI escape sequences from a string.

```go
func StripANSI(in string) string
```

### ValidateJobRequirements

ValidateJobRequirements checks that all required variables are present in the context.
Returns an error with a clear message listing missing variables.

```go
func ValidateJobRequirements(ctx *ExecutionContext, job *model.Job) error
```

### VisualLength

VisualLength returns the visual length of a string (excluding ANSI sequences).

```go
func VisualLength(s string) int
```

### Clone

Clone creates a copy with the same resolver.

```go
func (*ContextVariables) Clone() model.VariableStorage
```

### Get

Get returns a variable value, evaluating lazily if pending.

```go
func (*ContextVariables) Get(key string) any
```

### GetWithError

GetWithError returns a variable value and any lazy evaluation error.

```go
func (*ContextVariables) GetWithError(key string) (any, error)
```

### IsResolved

IsResolved reports whether a variable has completed lazy evaluation.

```go
func (*ContextVariables) IsResolved(key string) bool
```

### ResolveAll

ResolveAll evaluates all pending promises and returns the first error encountered.

```go
func (*ContextVariables) ResolveAll() error
```

### Set

Set stores a value directly (already resolved).

```go
func (*ContextVariables) Set(key string, value any)
```

### SetResolver

SetResolver updates the resolver function. Used when the resolver
needs a reference to the ContextVariables itself (circular setup).

```go
func (*ContextVariables) SetResolver(resolver func(string) (string, error))
```

### Walk

Walk iterates over all resolved values.

```go
func (*ContextVariables) Walk(fn func(key string, value any))
```

### Copy

Copy copies everything except Context. Variables are cloned.
jobTracker is shared (not copied) to maintain consistent dependency tracking.

```go
func (*ExecutionContext) Copy() *ExecutionContext
```

### EmitProgress

EmitProgress sends a job progress event if an observer is set.

```go
func (*ExecutionContext) EmitProgress(ev JobProgressEvent)
```

### IsJobCompleted

IsJobCompleted checks if a job has been completed.

```go
func (*ExecutionContext) IsJobCompleted(jobName string) bool
```

### MarkJobCompleted

MarkJobCompleted marks a job as completed.

```go
func (*ExecutionContext) MarkJobCompleted(jobName string)
```

### NextStepIndex

NextStepIndex returns the next sequential step index for this job execution.
This ensures each step/iteration gets a unique number.

```go
func (*ExecutionContext) NextStepIndex() int
```

### Render

Render refreshes the treeview.

```go
func (*ExecutionContext) Render()
```

### Resolve

Resolve resolves a task name using skill-local scope first, then global.
If the task has a ":" prefix, it resolves in global scope only (strict).

```go
func (*ExecutionContext) Resolve(taskName string) (*model.ResolvedTask, error)
```

### Resolver

Resolver provides task resolution in the execution context.

```go
func (*ExecutionContext) Resolver() *TaskResolver
```

### SkillResolver

SkillResolver provides task resolution in skill context.

```go
func (*ExecutionContext) SkillResolver() *TaskResolver
```

### ExecuteJob

ExecuteJob runs a single job.

```go
func (*Executor) ExecuteJob(parentCtx context.Context, execCtx *ExecutionContext) error
```

### Error

Error returns the error as a user message.

```go
func (*FuzzyMatchError) Error() string
```

### GetLines

GetLines returns all captured output as lines.

```go
func (*LineCapturingWriter) GetLines() []string
```

### String

String returns the raw captured output.

```go
func (*LineCapturingWriter) String() string
```

### Write

Write implements io.Writer.

```go
func (*LineCapturingWriter) Write(p []byte) (int, error)
```

### Lint

Lint validates the pipeline and returns any errors.

```go
func (*Linter) Lint() []LintError
```

### ID

ID returns the skill namespace, the file name without its extension.

```go
func (*Skill) ID() string
```

### AddSkillsDir

AddSkillsDir adds an additional skills directory to search.
Directories added later have lower precedence.

```go
func (*SkillsLoader) AddSkillsDir(dir string)
```

### FindFile

FindFile searches for files matching any of the given patterns starting from
startDir and traversing parent directories. Returns (found, matchDir) where
matchDir is the directory containing the first matched file.

For each directory (starting with startDir, going up), all patterns are checked.
This means closer matches are preferred over pattern order.

```go
func (*SkillsLoader) FindFile(patterns []string, startDir string) (string, bool)
```

### FindFolder

FindFolder searches for a directory with the given name starting from startDir
and traversing parent directories. Returns (found, containingDir) where
containingDir is the parent directory that contains the named folder.

```go
func (*SkillsLoader) FindFolder(name, startDir string) (string, bool)
```

### Load

Load discovers and returns all enabled skill pipelines.

A skill from an earlier directory shadows one of the same ID found
later, so a project skill wins over the global one it shares a name
with. A skill whose when: block does not match contributes nothing and
does not shadow anything.

```go
func (*SkillsLoader) Load() ([]*model.Pipeline, error)
```

### Scan

Scan returns every skill file in the loader's directories, in
directory order, whether or not its when: block is satisfied.

Load drops the skills that do not apply here; Scan keeps them so
`atkins --help` can list what is installed rather than only what the
current directory activates. Two directories carrying the same skill
ID both appear, the higher-priority one first.

```go
func (*SkillsLoader) Scan() ([]*Skill, error)
```

### Resolve

Resolve resolves a task name to its pipeline and job.

```go
func (*TaskResolver) Resolve(taskName string) (*model.ResolvedTask, error)
```

### ResolveName

ResolveName resolves a name to the pipeline and job.
It tries explicit matching, then checks aliases, then fuzzy matches jobs.
If no job is matched, an error is returned.

```go
func (*TaskResolver) ResolveName(name string, strict bool) (*model.ResolvedTask, error)
```

### ResolveWithFallback

ResolveWithFallback tries resolving with r first, then falls back to the
fallback resolver. If the task has a ":" prefix, only the fallback (global)
resolver is used.

```go
func (*TaskResolver) ResolveWithFallback(taskName string, fallback *TaskResolver) (*model.ResolvedTask, error)
```

### Pending

Pending returns the IDs a write would create or overwrite, leaving
out the skills already vendored unchanged.

```go
func (*VendorResult) Pending() []string
```

### Install

Install copies the selected skills into TargetDir, creating it when it
doesn't exist, and overwrites a vendored copy that has drifted from
its source. Vendored skills are tracked files like any other, so
reverting an unwanted change is the user's call, not this command's.

```go
func (*Vendorer) Install(result *VendorResult) error
```

### Plan

Plan reads the source skills and decides which of them this
repository uses, without writing anything.

A skill is used when its when: block matches a path anywhere in the
repository, when it has no when: block at all (it is active
everywhere), or when a pipeline in the repository - or another
selected skill - references one of its jobs.

```go
func (*Vendorer) Plan() (*VendorResult, error)
```

### Run

Run plans the vendoring and installs what it selected.

Plan on its own is the dry run: it reports the same selection without
touching the repository.

```go
func (*Vendorer) Run() (*VendorResult, error)
```

### Environ

Environ returns the environment as a slice of KEY=VALUE strings.

```go
func (Env) Environ() []string
```

### Sanitized

Sanitized returns a copy of the environment with atkins' own settings
removed.

It is what an agent hands to a command that arrived with a job: the
command keeps PATH and the rest of the machine's environment, and the
variables it is meant to receive are set by name afterwards. Removing
the prefixes wholesale keeps a setting added later with the process
that was configured with it.

```go
func (Env) Sanitized() Env
```

### Error

Error implements the error interface.

```go
func (ExecError) Error() string
```

### Len

Len returns the length of the error output.

```go
func (ExecError) Len() int
```

### OnJobProgress

OnJobProgress implements ProgressObserver.

```go
func (ProgressObserverFunc) OnJobProgress(ev JobProgressEvent)
```
