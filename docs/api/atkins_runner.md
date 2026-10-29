# Package ./runner

```go
import (
	"github.com/titpetric/atkins/runner"
)
```

Package `runner` loads atkins pipelines and runs their jobs.

A run starts from a pipeline file: DiscoverConfig finds it by walking up
from the working directory, LoadPipeline parses it, and a SkillsLoader
adds the skill pipelines from the .atkins/skills folders that apply. A
skill applies when its when: block matches a file near the working
directory, or when it has no when: block.

RunPipeline runs the requested jobs. Before any job starts, the pipeline
resolves the jobs and their dependencies into a run order and builds a
tree node for every job a step can invoke. The first render of the tree
shows the whole run. An Executor then runs each job's steps, with
their for: loops, task: references and conditions, and the tree and the
event log record what each step did.

Values reach a command through interpolation: `${{ expression }}` reads
variables and expressions, and `$(command)` substitutes the output of a
shell command. ContextVariables holds the variables of a run and can
resolve a pending one on its first read.

A project cloned inside a workspace checkout sees the workspace's skills
through SkillsLoader.AddParentWorkspaces. Inside home that walk stops
below home: $HOME/.atkins/skills loads on its own as the global skills,
and the folders above home are not the user's to configure. Outside home
the walk goes on to the filesystem root, the same as DiscoverConfig.

A task reference resolves through a TaskResolver: a skill's own jobs
first, then every loaded pipeline, with a leading colon skipping the
skill. The Vendorer copies the skills a repository uses into the
repository's own .atkins/skills.

## Types

<details>
<summary><code>type ContextVariables</code></summary>

```go
// ContextVariables holds variables in thread-safe storage with Promise-based lazy evaluation.
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
// Env is the environment a command runs with, read by NewEnv from the
// KEY=VALUE strings os.Environ and exec.Cmd use. For a duplicate name Env
// keeps the later entry, as exec does, and an entry without a name drops out.
//
// Sanitized drops atkins' own ATKINS_ and PLATFORM_ settings, for an agent
// that runs a job's command with the machine's environment and sets that
// command's variables by name. Dropping whole prefixes keeps a setting added
// later with the process that owns it.
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
// ExecutionContext holds the state of a run as a job and its steps see it.
// A job, a step and a loop iteration each run in a copy of their parent's
// context. Copy clones the variables and the environment, and a step's vars
// stay in the step. It shares the job tracker and the substitution memo, and
// every copy agrees on which jobs finished and on what a `$(...)`
// produced. A task name resolves in the skill's own pipeline first, then in
// every loaded pipeline; a leading colon skips the skill.
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

	// substitutions memoises the `$(...)` results of one step execution.
	// The passes that interpolate a step - the label, the condition,
	// the command - run the shell inside it once between them. Shared
	// across copies, and replaced for each step and each loop iteration.
	substitutions *substitutionCache

	// Progress receives job lifecycle events (optional).
	Progress ProgressObserver

	// Parents is the ancestor job chain for nested task invocations.
	Parents []string

	// Args are the command line arguments after a bare --. The exec:
	// steps of a job with args: true append them to their argv.
	Args []string
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
// FuzzyMatchError reports a pattern with more than one fuzzy match.
type FuzzyMatchError struct {
	Matches []*model.ResolvedTask
}
```

</details>

<details>
<summary><code>type IterationContext</code></summary>

```go
// IterationContext holds the variables for a single iteration of a for loop.
// ExpandFor returns one per iteration. An iterator reads "item in items",
// where items names a variable, "(index, item) in items" or
// "(key, value) in items", and any of them can take its items from a shell
// command: "item in $(ls ./bin/*.test)".
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
// Options holds configuration for the executor.
type Options struct {
	DefaultTimeout time.Duration
}
```

</details>

<details>
<summary><code>type OutputError</code></summary>

```go
// OutputError is the list output for a pipeline that fails the linter.
type OutputError struct {
	Error string            `json:"error" yaml:"error"`
	Lint  []OutputLintError `json:"lint" yaml:"lint"`
}
```

</details>

<details>
<summary><code>type OutputItem</code></summary>

```go
// OutputItem represents a single command in the list output.
type OutputItem struct {
	ID string `json:"id" yaml:"id"`

	// Name is the job name inside its pipeline, without the skill
	// prefix, or the alias itself.
	Name string `json:"name" yaml:"name"`

	Desc string `json:"desc,omitempty" yaml:"desc,omitempty"`
	Cmd  string `json:"cmd" yaml:"cmd"`

	// Interactive reports a job that reads from the terminal.
	Interactive bool `json:"interactive" yaml:"interactive"`

	// Args reports a job that takes arguments after a bare --.
	Args bool `json:"args" yaml:"args"`

	// Exec is the argv of the first exec: step of a job that takes
	// arguments, as the file writes it: the command the arguments
	// follow, for a caller to show before it runs the job.
	Exec []string `json:"exec,omitempty" yaml:"exec,omitempty"`

	// Aliases are the other names the job answers to.
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`

	// DependsOn names the jobs that run before this one.
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`

	// Steps are the step labels, the lines `atkins -l` prints under the
	// job. A caller can show what a job does without reading the
	// file itself.
	Steps []string `json:"steps,omitempty" yaml:"steps,omitempty"`

	// Env are the dotenv files the job reads on top of the pipeline's.
	Env []string `json:"env,omitempty" yaml:"env,omitempty"`

	// Hidden reports a job with show: false, a job the file keeps out of
	// menus. The listing carries it. A caller knows the job exists and
	// what it does, and decides for itself whether to show it.
	Hidden bool `json:"hidden,omitempty" yaml:"hidden,omitempty"`

	// Active is false for a job of an inactive skill, which only a
	// listing with --all includes. A job whose own when: block does not
	// match never lists.
	Active bool `json:"active" yaml:"active"`

	When *OutputWhen `json:"when,omitempty" yaml:"when,omitempty"`
}
```

</details>

<details>
<summary><code>type OutputLintError</code></summary>

```go
// OutputLintError is one linter finding in OutputError.
type OutputLintError struct {
	Pipeline string `json:"pipeline" yaml:"pipeline"`
	Path     string `json:"path,omitempty" yaml:"path,omitempty"`
	Job      string `json:"job" yaml:"job"`
	Issue    string `json:"issue" yaml:"issue"`
	Detail   string `json:"detail" yaml:"detail"`
}
```

</details>

<details>
<summary><code>type OutputOptions</code></summary>

```go
// OutputOptions adjusts the list output.
type OutputOptions struct {
	// All includes the skills whose when: block does not match, with
	// active: false.
	All bool

	// Dir is the folder the main pipeline runs in when it sets no
	// dir: of its own. Empty means the folder of its file.
	Dir string
}
```

</details>

<details>
<summary><code>type OutputSection</code></summary>

```go
// OutputSection represents a pipeline section in the list output.
type OutputSection struct {
	// ID is the skill namespace, empty for the main pipeline and the
	// aliases.
	ID string `json:"id" yaml:"id"`

	// Kind is one of pipeline, skill or aliases.
	Kind string `json:"kind" yaml:"kind"`

	// Path is the file the section loads from, empty for the aliases
	// and for a pipeline read from stdin.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Dir is the folder the section's jobs run in, empty for the
	// aliases and for an inactive skill.
	Dir string `json:"dir,omitempty" yaml:"dir,omitempty"`

	// Active is false for a skill whose when: block does not match.
	// Only a listing with --all includes one.
	Active bool `json:"active" yaml:"active"`

	When *OutputWhen `json:"when,omitempty" yaml:"when,omitempty"`

	Desc string `json:"desc" yaml:"desc"`

	// Help is the pipeline's help: block, the purpose it states.
	Help string `json:"help,omitempty" yaml:"help,omitempty"`

	// Include are the files the pipeline reads into its variables, and
	// Env the dotenv files it reads into the environment.
	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	Env     []string `json:"env,omitempty" yaml:"env,omitempty"`

	Cmds []OutputItem `json:"cmds" yaml:"cmds"`
}
```

</details>

<details>
<summary><code>type OutputWhen</code></summary>

```go
// OutputWhen reports a when: block in the list output and the file that
// satisfied it.
type OutputWhen struct {
	Files    []string `json:"files" yaml:"files"`
	Here     bool     `json:"here,omitempty" yaml:"here,omitempty"`
	Contains string   `json:"contains,omitempty" yaml:"contains,omitempty"`
	Matched  string   `json:"matched,omitempty" yaml:"matched,omitempty"`
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
	// terminal ends with. A caller uses it to record the run
	// somewhere else - a CI job log - without changing what a person
	// watching it sees.
	Transcript io.Writer

	// Args are the command line arguments after a bare --, handed to
	// the requested jobs. Only the exec: steps of a job with args: true
	// use them.
	Args []string

	// Recorder stores the finished run. Nil records nothing, which is
	// what a test and a linting pass want; the command line hands one
	// over so every run leaves a row behind.
	Recorder Recorder
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
<summary><code>type Recorder</code></summary>

```go
// Recorder stores what a finished run did. The event log document is
// the whole of it: the metadata, the tree, every event and the summary.
//
// The interface keeps the runner free of a database: a sqlite file, a
// socket or a test double can each stand behind it, and a recorder that fails
// never fails the run that produced it.
type Recorder interface {
	Record(ctx context.Context, log *eventlog.Log) error
}
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
// carry a markdown companion of the same base name, go.md, which holds a
// usage guide and no job list; `atkins --help` prints it under
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
	// its when: block, or the workspace root when it has none. The loader
	// resolves it only for an active skill.
	Dir string

	// Active reports whether the skill's when: block matches from
	// the directory the scan started in. Only an active skill
	// contributes its jobs to a run.
	Active bool

	// Matched is the file that satisfied the when: block, empty for a
	// skill without one or one that is not active.
	Matched string
}
```

</details>

<details>
<summary><code>type SkillsLoader</code></summary>

```go
// SkillsLoader loads the skill pipelines of .atkins/skills/ directories,
// in priority order: for each ID the first directory that holds an active
// skill supplies it, and the project directory comes first, the global one
// last. A skill runs in the folder of the file its when: block matched, or
// without one, in the folder holding its .atkins/. A skill whose when: block
// does not match is inactive and leaves the ID to the next directory.
//
// Load returns the active skills, the ones a run uses. Scan returns every
// skill for `atkins --help`, and LoadAll adds the inactive skills of an ID
// no active skill carries, for `atkins -l --json --all`.
type SkillsLoader struct {
	// SkillsDirs are the directories to search for skill files, in priority order.
	// The first directory holding an ID supplies that skill.
	SkillsDirs []string

	// StartDir is the directory from which to start searching for when: files.
	// This is typically the user's working directory.
	StartDir string

	// WorkspaceDir is the folder containing .atkins/ (used for skills without when:).
	WorkspaceDir string

	// Owners maps a skills directory to the folder holding its .atkins/,
	// where a skill without when: from that directory runs. A directory
	// missing from the map runs its skills in WorkspaceDir.
	Owners map[string]string
}
```

</details>

<details>
<summary><code>type TaskResolver</code></summary>

```go
// TaskResolver resolves a task reference to a pipeline and a job.
// A name resolves by exact name first, a qualified one such as go:build
// included, then by a job alias, then by a unique suffix match. A leading
// colon makes the reference strict: exact names only, and in a skill, the
// global pipelines only. A suffix that matches more than one job is a
// FuzzyMatchError.
type TaskResolver struct {
	pipelines []*model.Pipeline
}
```

</details>

<details>
<summary><code>type VarPromise</code></summary>

```go
// VarPromise represents a lazy variable that resolves on first access.
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
	// SourceDir is the skills directory Plan reads.
	SourceDir string

	// TargetDir is the repository directory Install writes to.
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
// VendorSkill is a candidate skill file, with the reason Plan picks it
// for vendoring once it matches against a repository.
type VendorSkill struct {
	// ID is the skill namespace, taken from the file name.
	ID string

	// Path is the absolute path of the source skill file.
	Path string

	// DocPath is the markdown companion beside Path, empty when the
	// skill carries none. Install copies it alongside the skill file.
	DocPath string

	// Pipeline is the parsed skill.
	Pipeline *model.Pipeline

	// Reason records why the repository uses the skill, e.g. the path that
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
// directory and into the repository. A clone or a CI agent gets the same
// jobs without a personal $HOME/.atkins/skills.
//
// A repository uses a skill whose when: block matches a path in it, a skill
// without a when: block, and a skill whose jobs its pipelines or its other
// skills reference. Plan reports that selection; Run also installs it, over
// a vendored copy that differs, which git tracks like any other file.
type Vendorer struct {
	// SourceDir is the skills directory to vendor from, typically
	// $HOME/.atkins/skills.
	SourceDir string

	// Root is the repository root, the folder holding .git.
	Root string

	// TargetDir is where Install writes skills, Root/.atkins/skills.
	TargetDir string

	// SkipDirs are directory names not descended into.
	SkipDirs []string
}
```

</details>

<details>
<summary><code>type WhenMatch</code></summary>

```go
// WhenMatch is the outcome of evaluating a when: block.
// An absolute pattern resolves on its own. A relative pattern resolves in
// the starting folder and then in every parent folder, nearest first, unless
// the block sets here: true, which keeps the search to the starting folder.
// In each folder the search tries every pattern before it moves to the
// parent folder, and the nearest folder with a match decides. With contains: set, only a regular file whose contents match the
// regular expression counts. An invalid contains: expression never matches;
// the linter reports it.
type WhenMatch struct {
	// Dir is the folder holding the matched file, where a skill runs.
	Dir string

	// File is the path the matching pattern resolved to.
	File string
}
```

</details>

## Consts

<details>
<summary><code>const KindPipeline, KindSkill, KindAliases</code></summary>

```go
// Section kinds in the list output.
const (
	KindPipeline = "pipeline"
	KindSkill    = "skill"
	KindAliases  = "aliases"
)
```

</details>

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
// The states Plan sets on a used skill.
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
// ConfigNames are the default configuration file names to search for, in order of preference.
var ConfigNames = []string{".atkins.yml", ".atkins.yaml", "atkins.yml", "atkins.yaml"}
```

</details>

<details>
<summary><code>var ErrJobSkipped</code></summary>

```go
// ErrJobSkipped reports a job whose if condition evaluates to false.
var ErrJobSkipped = errors.New("job skipped")
```

</details>

## Function symbols

- `func ActivePipelines (pipelines []*model.Pipeline) []*model.Pipeline`
- `func ApplyJobWhen (pipelines []*model.Pipeline, startDir string)`
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
- `func ListPipelinesJSON (pipelines []*model.Pipeline, opts ...OutputOptions) error`
- `func ListPipelinesYAML (pipelines []*model.Pipeline, opts ...OutputOptions) error`
- `func LoadPipeline (filePath string) ([]*model.Pipeline, error)`
- `func LoadPipelineFromReader (r io.Reader) ([]*model.Pipeline, error)`
- `func MatchWhen (when *model.PipelineWhen, startDir string) (WhenMatch, bool)`
- `func MatchWhenPattern (dir,pattern string) (string, bool)`
- `func MergeSkillVariables (ctx *ExecutionContext, decl *model.Decl) error`
- `func MergeSkills (pipelines,more []*model.Pipeline) []*model.Pipeline`
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
- `func NewOutputError (pipeline *model.Pipeline, lintErrors []LintError) OutputError`
- `func NewPipeline (data *model.Pipeline, opts PipelineOptions) *Pipeline`
- `func NewSkillResolver (pipeline *model.Pipeline) *TaskResolver`
- `func NewSkillsLoader (workspaceDir,startDir string) *SkillsLoader`
- `func NewTaskResolver (pipelines []*model.Pipeline) *TaskResolver`
- `func NewVendorer (sourceDir,root string) *Vendorer`
- `func ParentWorkspaces (dir,home string) []string`
- `func PrintJSON (v any) error`
- `func PrintYAML (v any) error`
- `func ProcessDecl (ctx *ExecutionContext, decl *model.Decl) (map[string]any, error)`
- `func ResolveJobDependencies (jobs map[string]*model.Job, startingJob string) ([]string, error)`
- `func RunPipeline (ctx context.Context, pipeline *model.Pipeline, opts PipelineOptions) error`
- `func Sanitize (in string) ([]string, error)`
- `func StripANSI (in string) string`
- `func ValidateJobRequirements (ctx *ExecutionContext, job *model.Job) error`
- `func ValidateWhen (when *model.PipelineWhen) error`
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
- `func (*Pipeline) Run (ctx context.Context) error`
- `func (*Skill) ID () string`
- `func (*SkillsLoader) AddParentWorkspaces ()`
- `func (*SkillsLoader) AddSkillsDir (dir string)`
- `func (*SkillsLoader) AddWorkspace (dir string)`
- `func (*SkillsLoader) FindFile (patterns []string, startDir string) (string, bool)`
- `func (*SkillsLoader) FindFolder (name,startDir string) (string, bool)`
- `func (*SkillsLoader) Load () ([]*model.Pipeline, error)`
- `func (*SkillsLoader) LoadAll () ([]*model.Pipeline, error)`
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

### `ActivePipelines`

`ActivePipelines` returns the pipelines a run sees: the active skills with their active jobs.

```go
func ActivePipelines(pipelines []*model.Pipeline) []*model.Pipeline
```

### `ApplyJobWhen`

`ApplyJobWhen` marks Inactive each job whose when: block does not match from startDir.

```go
func ApplyJobWhen(pipelines []*model.Pipeline, startDir string)
```

### `DefaultOptions`

`DefaultOptions` returns the default executor options.

```go
func DefaultOptions() *Options
```

### `DiscoverConfig`

`DiscoverConfig` searches for a configuration file starting from the given directory,
traversing parent directories until it finds a configuration file or reaches root.
Returns the absolute path to the configuration file and the directory containing it.

```go
func DiscoverConfig(startDir string) (string, error)
```

### `DiscoverConfigFromCwd`

`DiscoverConfigFromCwd` is a convenience wrapper that starts from the current working directory.

```go
func DiscoverConfigFromCwd() (string, error)
```

### `DiscoverEnvironment`

`DiscoverEnvironment` scans for marker files starting from startDir,
traversing parent directories up to the filesystem root.
Root holds the highest directory that contains any markers.

```go
func DiscoverEnvironment(startDir string) (*Environment, error)
```

### `DiscoverEnvironmentFromCwd`

`DiscoverEnvironmentFromCwd` is a convenience wrapper that starts from the current working directory.

```go
func DiscoverEnvironmentFromCwd() (*Environment, error)
```

### `EvaluateIf`

`EvaluateIf` evaluates the If condition using expr-lang.
Returns true if the condition holds, false if no condition or condition is false.
With more than one condition, all must be true (AND logic).
Returns error only for invalid expressions.

```go
func EvaluateIf(ctx *ExecutionContext) (bool, error)
```

### `EvaluateJobIf`

`EvaluateJobIf` evaluates the If condition on a job using expr-lang.
Returns true if the condition holds or the job has no condition.
With more than one condition, all must be true (AND logic).
Returns error only for invalid expressions.

```go
func EvaluateJobIf(ctx *ExecutionContext) (bool, error)
```

### `ExpandFor`

`ExpandFor` returns one context per iteration of the step's for: loop,
nil for a step without one, and the error of an iterator that does not expand.
More than one iterator returns their cartesian product; see IterationContext
for the patterns an iterator takes.

```go
func ExpandFor(ctx *ExecutionContext, executeCommand func(string) (string, error)) ([]IterationContext, error)
```

### `FindRepositoryRoot`

`FindRepositoryRoot` returns the folder holding .git, starting at
startDir and traversing parent directories.

```go
func FindRepositoryRoot(startDir string) (string, error)
```

### `GetDependencies`

`GetDependencies` converts depends_on field (string or []string) to a slice of job names.

```go
func GetDependencies(dependsOn any) []string
```

### `InterpolateCommand`

`InterpolateCommand` interpolates a command string.

```go
func InterpolateCommand(cmd string, ctx *ExecutionContext) (string, error)
```

### `InterpolateMap`

`InterpolateMap` recursively interpolates all string values in a map.

```go
func InterpolateMap(ctx *ExecutionContext, m map[string]any) error
```

### `InterpolateString`

`InterpolateString` returns s with each ${{ expression }} replaced: a variable, a dotted path, or an expr expression with ?? and ||.

```go
func InterpolateString(s string, ctx *ExecutionContext) (string, error)
```

### `IsEchoCommand`

`IsEchoCommand` checks if a command is a bare echo command.

```go
func IsEchoCommand(cmd string) bool
```

### `ListPipelines`

`ListPipelines` returns pipelines formatted as a string in a flat list format:
Main Pipeline, then Aliases, then Skills.

```go
func ListPipelines(pipelines []*model.Pipeline) string
```

### `ListPipelinesJSON`

`ListPipelinesJSON` outputs pipelines in JSON format.

```go
func ListPipelinesJSON(pipelines []*model.Pipeline, opts ...OutputOptions) error
```

### `ListPipelinesYAML`

`ListPipelinesYAML` outputs pipelines in YAML format.

```go
func ListPipelinesYAML(pipelines []*model.Pipeline, opts ...OutputOptions) error
```

### `LoadPipeline`

`LoadPipeline` loads and parses a pipeline from a yaml file.
Returns the number of documents loaded, the parsed pipeline, and any error.

```go
func LoadPipeline(filePath string) ([]*model.Pipeline, error)
```

### `LoadPipelineFromReader`

`LoadPipelineFromReader` loads and parses a pipeline from an io.Reader.
Returns the parsed pipeline(s) and any error.

```go
func LoadPipelineFromReader(r io.Reader) ([]*model.Pipeline, error)
```

### `MatchWhen`

`MatchWhen` reports false for an empty block or an invalid contains: expression,
and otherwise the first match from startDir, absolute patterns first.

```go
func MatchWhen(when *model.PipelineWhen, startDir string) (WhenMatch, bool)
```

### `MatchWhenPattern`

`MatchWhenPattern` returns what a `when: files:` pattern resolves to inside dir.
A glob pattern such as `schema/*.up.sql` matches a folder's contents, `schema/` the folder.

```go
func MatchWhenPattern(dir, pattern string) (string, bool)
```

### `MergeSkillVariables`

`MergeSkillVariables` merges variables from a skill's Decl into the context.
When depth > 1, variables are already on the stack from a parent pipeline.
The skill's vars fill in missing keys and keep the existing values.
At depth <= 1 it behaves identically to MergeVariables.

```go
func MergeSkillVariables(ctx *ExecutionContext, decl *model.Decl) error
```

### `MergeSkills`

`MergeSkills` returns pipelines with the new IDs of more appended, an active one replacing an inactive one.

```go
func MergeSkills(pipelines, more []*model.Pipeline) []*model.Pipeline
```

### `MergeVariables`

`MergeVariables` merges variables from Decl into the execution context.
When both vars and env.vars are present, it resolves them together using
a unified dependency graph so that cross-references work correctly
(e.g., vars using $(echo $ENV_VAR) and env using ${{ var_name }}).

```go
func MergeVariables(ctx *ExecutionContext, decl *model.Decl) error
```

### `NewContextVariables`

`NewContextVariables` creates a ContextVariables from a map of evaluated values.

```go
func NewContextVariables(values map[string]any) *ContextVariables
```

### `NewContextVariablesWithResolver`

`NewContextVariablesWithResolver` returns variables that resolve on their first Get.

```go
func NewContextVariablesWithResolver(pending map[string]any, resolver func(string) (string, error)) *ContextVariables
```

### `NewEnv`

`NewEnv` returns the Env the KEY=VALUE entries describe.

```go
func NewEnv(environ []string) Env
```

### `NewExecError`

`NewExecError` creates an ExecError from a psexec.Result.

```go
func NewExecError(result psexec.Result) ExecError
```

### `NewExecutor`

`NewExecutor` creates a new executor with default options.

```go
func NewExecutor() *Executor
```

### `NewExecutorWithOptions`

`NewExecutorWithOptions` creates a new executor with custom options.

```go
func NewExecutorWithOptions(opts *Options) *Executor
```

### `NewLineCapturingWriter`

`NewLineCapturingWriter` creates a new LineCapturingWriter.

```go
func NewLineCapturingWriter() *LineCapturingWriter
```

### `NewLinter`

`NewLinter` creates a new linter.

```go
func NewLinter(pipeline *model.Pipeline) *Linter
```

### `NewLinterWithPipelines`

`NewLinterWithPipelines` creates a linter with access to all pipelines for cross-pipeline validation.

```go
func NewLinterWithPipelines(pipeline *model.Pipeline, allPipelines []*model.Pipeline) *Linter
```

### `NewOutputError`

`NewOutputError` builds the list output for lint findings in a pipeline.

```go
func NewOutputError(pipeline *model.Pipeline, lintErrors []LintError) OutputError
```

### `NewPipeline`

`NewPipeline` allocates a new *Pipeline with dependencies.

```go
func NewPipeline(data *model.Pipeline, opts PipelineOptions) *Pipeline
```

### `NewSkillResolver`

`NewSkillResolver` returns a task resolver for a skill pipeline.

```go
func NewSkillResolver(pipeline *model.Pipeline) *TaskResolver
```

### `NewSkillsLoader`

`NewSkillsLoader` returns a loader for workspaceDir/.atkins/skills, matching when: blocks from startDir.

```go
func NewSkillsLoader(workspaceDir, startDir string) *SkillsLoader
```

### `NewTaskResolver`

`NewTaskResolver` returns a task resolver for a set of pipelines.

```go
func NewTaskResolver(pipelines []*model.Pipeline) *TaskResolver
```

### `NewVendorer`

`NewVendorer` creates a vendorer writing into root/.atkins/skills.

```go
func NewVendorer(sourceDir, root string) *Vendorer
```

### `ParentWorkspaces`

`ParentWorkspaces` returns the folders above dir holding .atkins/skills, nearest first,
stopping below home when dir is inside it.

```go
func ParentWorkspaces(dir, home string) []string
```

### `PrintJSON`

`PrintJSON` writes v as indented JSON to stdout.

```go
func PrintJSON(v any) error
```

### `PrintYAML`

`PrintYAML` writes v as YAML to stdout.

```go
func PrintYAML(v any) error
```

### `ProcessDecl`

`ProcessDecl` returns an error for an include file that does not load,
an error for a var that does not interpolate,
and otherwise the include files' values with the interpolated vars over them.

```go
func ProcessDecl(ctx *ExecutionContext, decl *model.Decl) (map[string]any, error)
```

### `ResolveJobDependencies`

`ResolveJobDependencies` returns jobs in dependency order.
Returns the jobs to run and any resolution errors.

```go
func ResolveJobDependencies(jobs map[string]*model.Job, startingJob string) ([]string, error)
```

### `RunPipeline`

`RunPipeline` runs a pipeline with the given options.

```go
func RunPipeline(ctx context.Context, pipeline *model.Pipeline, opts PipelineOptions) error
```

### `Sanitize`

`Sanitize` returns nil for empty input.
Otherwise it returns the lines a terminal shows, ANSI colours kept.

```go
func Sanitize(in string) ([]string, error)
```

### `StripANSI`

`StripANSI` removes all ANSI escape sequences from a string.

```go
func StripANSI(in string) string
```

### `ValidateJobRequirements`

`ValidateJobRequirements` checks that all required variables are present in the context.
Returns an error with a clear message listing missing variables.

```go
func ValidateJobRequirements(ctx *ExecutionContext, job *model.Job) error
```

### `ValidateWhen`

`ValidateWhen` reports a when: block that can never match because its
contains: expression does not compile.

```go
func ValidateWhen(when *model.PipelineWhen) error
```

### `VisualLength`

`VisualLength` returns the visual length of a string (excluding ANSI sequences).

```go
func VisualLength(s string) int
```

### `Clone`

`Clone` creates a copy with the same resolver.

```go
func (*ContextVariables) Clone() model.VariableStorage
```

### `Get`

`Get` returns a variable value, evaluating lazily if pending.

```go
func (*ContextVariables) Get(key string) any
```

### `GetWithError`

`GetWithError` returns a variable value and any lazy evaluation error.

```go
func (*ContextVariables) GetWithError(key string) (any, error)
```

### `IsResolved`

`IsResolved` reports whether a variable has completed lazy evaluation.

```go
func (*ContextVariables) IsResolved(key string) bool
```

### `ResolveAll`

`ResolveAll` evaluates all pending promises and returns the first error encountered.

```go
func (*ContextVariables) ResolveAll() error
```

### `Set`

`Set` stores a value directly (already resolved).

```go
func (*ContextVariables) Set(key string, value any)
```

### `SetResolver`

`SetResolver` sets the resolver, for one that reads these same variables.

```go
func (*ContextVariables) SetResolver(resolver func(string) (string, error))
```

### `Walk`

`Walk` iterates over all resolved values.

```go
func (*ContextVariables) Walk(fn func(key string, value any))
```

### `Copy`

`Copy` returns a copy one level deeper, without Context.

```go
func (*ExecutionContext) Copy() *ExecutionContext
```

### `EmitProgress`

`EmitProgress` sends a job progress event if the context has an observer.

```go
func (*ExecutionContext) EmitProgress(ev JobProgressEvent)
```

### `IsJobCompleted`

`IsJobCompleted` checks if a job has completed.

```go
func (*ExecutionContext) IsJobCompleted(jobName string) bool
```

### `MarkJobCompleted`

`MarkJobCompleted` marks a job as completed.

```go
func (*ExecutionContext) MarkJobCompleted(jobName string)
```

### `NextStepIndex`

`NextStepIndex` returns the next step index of the job, unique to each step and iteration.

```go
func (*ExecutionContext) NextStepIndex() int
```

### `Render`

`Render` refreshes the treeview.

```go
func (*ExecutionContext) Render()
```

### `Resolve`

`Resolve` returns the task a name resolves to, in the skill first, then globally.

```go
func (*ExecutionContext) Resolve(taskName string) (*model.ResolvedTask, error)
```

### `Resolver`

`Resolver` resolves tasks in the execution context.

```go
func (*ExecutionContext) Resolver() *TaskResolver
```

### `SkillResolver`

`SkillResolver` resolves tasks in skill context.

```go
func (*ExecutionContext) SkillResolver() *TaskResolver
```

### `ExecuteJob`

`ExecuteJob` runs a single job.

```go
func (*Executor) ExecuteJob(parentCtx context.Context, execCtx *ExecutionContext) error
```

### `Error`

`Error` returns the error as a user message.

```go
func (*FuzzyMatchError) Error() string
```

### `GetLines`

`GetLines` returns all captured output as lines.

```go
func (*LineCapturingWriter) GetLines() []string
```

### `String`

`String` returns the raw captured output.

```go
func (*LineCapturingWriter) String() string
```

### `Write`

`Write` implements io.Writer.

```go
func (*LineCapturingWriter) Write(p []byte) (int, error)
```

### `Lint`

`Lint` validates the pipeline and returns any errors.

```go
func (*Linter) Lint() []LintError
```

### `Run`

`Run` returns the error of the first job that fails, nil when every job passes.

```go
func (*Pipeline) Run(ctx context.Context) error
```

### `ID`

`ID` returns the skill namespace, the file name without its extension.

```go
func (*Skill) ID() string
```

### `AddParentWorkspaces`

`AddParentWorkspaces` adds the workspaces ParentWorkspaces finds above WorkspaceDir.

```go
func (*SkillsLoader) AddParentWorkspaces()
```

### `AddSkillsDir`

`AddSkillsDir` adds a skills directory below the ones already there.

```go
func (*SkillsLoader) AddSkillsDir(dir string)
```

### `AddWorkspace`

`AddWorkspace` adds dir/.atkins/skills, whose skills without when: run in dir.

```go
func (*SkillsLoader) AddWorkspace(dir string)
```

### `FindFile`

`FindFile` returns the nearest folder from startDir up where any pattern matches.
It checks every pattern in a folder before moving to the parent.

```go
func (*SkillsLoader) FindFile(patterns []string, startDir string) (string, bool)
```

### `FindFolder`

`FindFolder` returns the nearest folder from startDir up that holds a folder named name.

```go
func (*SkillsLoader) FindFolder(name, startDir string) (string, bool)
```

### `Load`

`Load` returns the error of a skill that does not load,
or the pipelines of the active skills, one per ID.

```go
func (*SkillsLoader) Load() ([]*model.Pipeline, error)
```

### `LoadAll`

`LoadAll` returns the error of a skill that does not load,
or the pipelines Load returns followed by the inactive skills, marked Inactive.

```go
func (*SkillsLoader) LoadAll() ([]*model.Pipeline, error)
```

### `Scan`

`Scan` returns every skill in directory order, active and inactive, with each ID's later copies,
or the error of a directory that does not read
or a skill that does not load.

```go
func (*SkillsLoader) Scan() ([]*Skill, error)
```

### `Resolve`

`Resolve` resolves a task name to its pipeline and job.

```go
func (*TaskResolver) Resolve(taskName string) (*model.ResolvedTask, error)
```

### `ResolveName`

`ResolveName` returns an exact match, else an alias, else a fuzzy match; strict stops after the exact one.

```go
func (*TaskResolver) ResolveName(name string, strict bool) (*model.ResolvedTask, error)
```

### `ResolveWithFallback`

`ResolveWithFallback` resolves a ":" name with fallback alone,
any other name with r first and then with fallback.

```go
func (*TaskResolver) ResolveWithFallback(taskName string, fallback *TaskResolver) (*model.ResolvedTask, error)
```

### `Pending`

`Pending` returns the IDs a write would create or overwrite.

```go
func (*VendorResult) Pending() []string
```

### `Install`

`Install` copies the pending skills into TargetDir, creating it when it
doesn't exist, and returns the first error that stops a copy.

```go
func (*Vendorer) Install(result *VendorResult) error
```

### `Plan`

`Plan` returns the skills the repository uses, or the error that stops it reading them.

```go
func (*Vendorer) Plan() (*VendorResult, error)
```

### `Run`

`Run` plans the vendoring and installs what it selected.

Plan on its own is the dry run: it reports the same selection without
touching the repository.

```go
func (*Vendorer) Run() (*VendorResult, error)
```

### `Environ`

`Environ` returns the environment as a slice of KEY=VALUE strings.

```go
func (Env) Environ() []string
```

### `Sanitized`

`Sanitized` returns nil for a nil Env,
otherwise a copy without atkins' own settings.

```go
func (Env) Sanitized() Env
```

### `Error`

`Error` implements the error interface.

```go
func (ExecError) Error() string
```

### `Len`

`Len` returns the length of the error output.

```go
func (ExecError) Len() int
```

### `OnJobProgress`

`OnJobProgress` implements ProgressObserver.

```go
func (ProgressObserverFunc) OnJobProgress(ev JobProgressEvent)
```
