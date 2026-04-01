# Package ./agent

```go
import (
	"github.com/titpetric/atkins/agent"
}
```

## Types

```go
// Agent manages the interactive REPL session.
type Agent struct {
	pipelines []*model.Pipeline
	resolver  *runner.TaskResolver
	options   *Options
	workDir   string
}
```

```go
// AliasEntry maps a natural language phrase to a task name.
type AliasEntry struct {
	Phrase string `yaml:"phrase"`
	Task   string `yaml:"task"`
}
```

```go
// AliasStore manages user-defined phrase → task corrections.
type AliasStore struct {
	Aliases []AliasEntry `yaml:"aliases"`
	path    string
}
```

```go
// AutoFixConfig holds configuration for auto-fix behavior.
type AutoFixConfig struct {
	// Enabled controls whether auto-fix is attempted.
	Enabled bool

	// MaxRetries is the maximum number of fix+retry cycles.
	MaxRetries int

	// FixTaskSuffix is the suffix used to find fix tasks (default: "fix").
	FixTaskSuffix string
}
```

```go
// AutoFixer handles automatic error recovery.
type AutoFixer struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline
}
```

```go
// Breadcrumb tracks execution progress as a one-liner.
type Breadcrumb struct {
	segments  []string
	status    string
	startTime time.Time
}
```

```go
// GitStats holds +/- line counts from git diff.
type GitStats struct {
	Added   int
	Removed int
}
```

```go
// Greeter handles greeting detection and responses.
type Greeter struct {
	groups []GreetingGroup
}
```

```go
// GreetingGroup maps a language/group to its trigger words and responses.
type GreetingGroup struct {
	Keywords  []string `yaml:"keywords"`
	Responses []string `yaml:"responses"`
}
```

```go
// GreetingsConfig is the YAML structure for ~/.atkins/greetings.yaml.
type GreetingsConfig struct {
	Greetings []GreetingGroup `yaml:"greetings"`
}
```

```go
// Intent represents a parsed user intent.
type Intent struct {
	Type     IntentType
	Raw      string              // Original input
	Keywords []string            // Extracted keywords
	Task     string              // Resolved task name (e.g., "go:test")
	Command  string              // Slash command name (without /)
	Args     string              // Arguments for slash command
	Resolved *model.ResolvedTask // Resolved task reference
}
```

```go
// IntentType categorizes user input.
type IntentType int
```

```go
// LogEntry represents a single entry in the message log.
type LogEntry struct {
	Time     time.Time
	Kind     string // "info", "error", "run", "prompt"
	Text     string
	Task     string
	Running  bool
	Started  time.Time
	Duration time.Duration
	Failed   bool
}
```

```go
// Model is the bubbletea model for the agent REPL.
type Model struct {
	agent      *Agent
	state      State
	input      string
	cursor     int
	history    []string
	historyIdx int
	breadcrumb *Breadcrumb
	lastError  error
	lastTask   *model.ResolvedTask
	retryCount int

	// Centralized router and slash commands
	router   *Router
	registry *Registry

	// Dimensions
	width  int
	height int

	// TUI state
	version   string
	hostname  string
	cwd       string
	gitBranch string
	gitStats  GitStats

	log       []LogEntry
	scrollOff int
	spinner   spinner.Model
	runLogIdx int // index of the current running entry in log

	// Confirmation state for fuzzy matching
	pendingConfirm *Route
}
```

```go
// Options configures agent behavior.
type Options struct {
	Debug   bool
	Verbose bool
	Jail    bool
}
```

```go
// Parser parses user input into intents.
type Parser struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline
	aliases  *AliasStore
}
```

```go
// Registry holds all registered slash commands.
type Registry struct {
	commands map[string]*SlashCommand
	ordered  []string
}
```

```go
// Route represents a routing decision.
type Route struct {
	Type        RouteType
	Raw         string              // Original input
	Task        string              // Task name for RouteTask
	Resolved    *model.ResolvedTask // Resolved task for RouteTask
	Command     string              // Slash command name for RouteSlash
	Args        string              // Arguments for RouteSlash
	ShellCmd    string              // Shell command for RouteShell
	Greeting    string              // Greeting response for RouteGreeting
	Fortune     string              // Fortune text for RouteFortune
	Phrase      string              // Phrase for RouteCorrection
	AliasTask   string              // Alias target for RouteCorrection
	Ambiguous   bool                // Multiple matches found
	Matches     []string            // Matching skills when ambiguous
	HistMatches []ShellHistoryEntry // Shell history matches

	// Multi-task support (chained commands)
	Tasks []*model.ResolvedTask // Multiple tasks for RouteMultiTask

	// Fuzzy match confirmation
	Suggestion string // Suggested correction for RouteConfirm
	Original   string // Original input that was fuzzy matched
}
```

```go
// RouteType categorizes the routing decision.
type RouteType int
```

```go
// Router implements the centralized routing logic based on structure.d2.
// Flow: Prompt → Is alias? → Semantic parsing → Match (skills/targets)?
type Router struct {
	resolver     *runner.TaskResolver
	skills       []*model.Pipeline
	aliases      *AliasStore
	greeter      *Greeter
	registry     *Registry
	shellHistory *ShellHistory

	// Context for retry/again
	lastInput  string // Last input for retry
	lastFailed bool   // Whether last command failed
}
```

```go
// ShellHistory maintains a persistent history of shell commands.
type ShellHistory struct {
	Entries []ShellHistoryEntry `json:"entries"`
	path    string
}
```

```go
// ShellHistoryEntry records a shell command execution.
type ShellHistoryEntry struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Dir      string        `json:"dir"`
	Time     time.Time     `json:"time"`
}
```

```go
// SlashCommand represents a slash command handler.
type SlashCommand struct {
	Name        string
	Aliases     []string
	Description string
	Handler     func(m *Model, args string) (Model, tea.Cmd)
}
```

```go
// State represents the current REPL state.
type State int
```

```go
// Messages for async operations.
type (
	ExecutionStartMsg struct {
		Input    string // original user input
		Task     string
		Resolved *model.ResolvedTask
	}
	ExecutionDoneMsg struct {
		Task     *model.ResolvedTask
		Err      error
		Duration time.Duration
	}
	AutofixStartMsg struct {
		OriginalTask *model.ResolvedTask
		FixTask      *model.ResolvedTask
	}
	AutofixDoneMsg struct {
		OriginalTask *model.ResolvedTask
		Err          error
		Duration     time.Duration
	}
	RetryMsg struct {
		Task *model.ResolvedTask
	}
	ShellStartMsg struct {
		Command string
	}
	ShellDoneMsg struct {
		Command  string
		Output   string
		Err      error
		ExitCode int
		Duration time.Duration
	}
)
```

## Consts

```go
// IntentType constants for the enum.
const (
	IntentUnknown IntentType = iota
	IntentTask               // Run a skill/task
	IntentSlash              // Slash command
	IntentHelp               // Help request
	IntentQuit               // Exit request
)
```

```go
// State constants for the REPL.
const (
	StateIdle State = iota
	StateExecuting
	StateAutofix
	StateRetrying
)
```

```go
// RouteType constants following structure.d2 flow.
const (
	RouteUnknown    RouteType = iota
	RouteAlias                // Resolved via alias
	RouteSlash                // Slash command (explicit or natural language)
	RouteTask                 // Skill/task execution
	RouteMultiTask            // Multiple tasks (chained with && or "then")
	RouteShell                // Shell command execution
	RouteGreeting             // Greeting response
	RouteCorrection           // Store a correction/alias
	RouteFortune              // Fortune/motivation request
	RouteHelp                 // Help request
	RouteQuit                 // Exit request
	RouteRetry                // Retry last command (again/retry)
	RouteConfirm              // Fuzzy match needs confirmation
)
```

## Vars

```go
// FillerWords to strip from natural language input.
var FillerWords = []string{
	"give", "me", "the", "a", "an", "please", "can", "you",
	"i", "want", "need", "get", "show", "run", "execute",
	"do", "make", "let", "lets", "let's", "my", "some",
	"what", "is", "are", "how", "about", "whats", "what's",
	"your", "its", "it's", "tell", "whats",
}
```

## Function symbols

- `func DefaultAutoFixConfig () *AutoFixConfig`
- `func DefaultRegistry () *Registry`
- `func Fortune () string`
- `func MatchFortune (input string) bool`
- `func New (workDir string, opts *Options) (*Agent, error)`
- `func NewAliasStore () *AliasStore`
- `func NewAutoFixer (resolver *runner.TaskResolver, skills []*model.Pipeline) *AutoFixer`
- `func NewBreadcrumb () *Breadcrumb`
- `func NewGreeter () *Greeter`
- `func NewModel (agent *Agent, version string) Model`
- `func NewParser (resolver *runner.TaskResolver, skills []*model.Pipeline) *Parser`
- `func NewRegistry () *Registry`
- `func NewRouter (resolver *runner.TaskResolver, skills []*model.Pipeline, registry *Registry) *Router`
- `func NewShellHistory () *ShellHistory`
- `func ParseCorrection (input string) (string, string, bool)`
- `func UsageText () string`
- `func (*Agent) Exec (ctx context.Context, prompt,version string) error`
- `func (*Agent) Options () *Options`
- `func (*Agent) Pipelines () []*model.Pipeline`
- `func (*Agent) Resolver () *runner.TaskResolver`
- `func (*Agent) Run (ctx context.Context, version string) error`
- `func (*Agent) WorkDir () string`
- `func (*AliasStore) Add (phrase,task string)`
- `func (*AliasStore) Match (input string) string`
- `func (*AutoFixer) AttemptFix (ctx context.Context, task *model.ResolvedTask, allPipelines []*model.Pipeline) error`
- `func (*AutoFixer) CanFix (task *model.ResolvedTask) bool`
- `func (*AutoFixer) GetFixTask (task *model.ResolvedTask) (*model.ResolvedTask, error)`
- `func (*Breadcrumb) Clear ()`
- `func (*Breadcrumb) LastSegment () string`
- `func (*Breadcrumb) Pop ()`
- `func (*Breadcrumb) Push (segment string)`
- `func (*Breadcrumb) SetStatus (status string)`
- `func (*Breadcrumb) String () string`
- `func (*Greeter) LearnGreeting (input string) (string, bool)`
- `func (*Greeter) Match (input string) string`
- `func (*Parser) Aliases () *AliasStore`
- `func (*Parser) AvailableSkills () []string`
- `func (*Parser) FindMatches (keywords []string) []string`
- `func (*Parser) Parse (input string) (*Intent, error)`
- `func (*Registry) Get (name string) *SlashCommand`
- `func (*Registry) HelpText () string`
- `func (*Registry) Register (cmd *SlashCommand)`
- `func (*Router) Aliases () *AliasStore`
- `func (*Router) AvailableSkills () []string`
- `func (*Router) FindMatches (keywords []string) []string`
- `func (*Router) Greeter () *Greeter`
- `func (*Router) LastCommand () string`
- `func (*Router) Route (input string) *Route`
- `func (*Router) SetLastCommand (input string, failed bool)`
- `func (*Router) ShellHistory () *ShellHistory`
- `func (*ShellHistory) Add (command string, exitCode int, duration time.Duration, dir string)`
- `func (*ShellHistory) FindExact (command string) *ShellHistoryEntry`
- `func (*ShellHistory) Match (input string) []ShellHistoryEntry`
- `func (Model) Init () tea.Cmd`
- `func (Model) Update (msg tea.Msg) (tea.Model, tea.Cmd)`
- `func (Model) View () tea.View`

### DefaultAutoFixConfig

DefaultAutoFixConfig returns the default auto-fix configuration.

```go
func DefaultAutoFixConfig() *AutoFixConfig
```

### DefaultRegistry

DefaultRegistry returns the built-in slash commands.

```go
func DefaultRegistry() *Registry
```

### Fortune

Fortune returns a fortune string. Uses the system `fortune` command if
available, otherwise returns a random coding motivational quote.

```go
func Fortune() string
```

### MatchFortune

MatchFortune returns true if the input is asking for a fortune.

```go
func MatchFortune(input string) bool
```

### New

New creates a new Agent with discovered skills.

```go
func New(workDir string, opts *Options) (*Agent, error)
```

### NewAliasStore

NewAliasStore loads or creates the alias store.

```go
func NewAliasStore() *AliasStore
```

### NewAutoFixer

NewAutoFixer creates a new auto-fixer.

```go
func NewAutoFixer(resolver *runner.TaskResolver, skills []*model.Pipeline) *AutoFixer
```

### NewBreadcrumb

NewBreadcrumb creates a new breadcrumb tracker.

```go
func NewBreadcrumb() *Breadcrumb
```

### NewGreeter

NewGreeter creates a greeter with built-in defaults merged with user config.

```go
func NewGreeter() *Greeter
```

### NewModel

NewModel creates a new bubbletea model for the agent.

```go
func NewModel(agent *Agent, version string) Model
```

### NewParser

NewParser creates a new intent parser.

```go
func NewParser(resolver *runner.TaskResolver, skills []*model.Pipeline) *Parser
```

### NewRegistry

NewRegistry creates a new slash command registry.

```go
func NewRegistry() *Registry
```

### NewRouter

NewRouter creates a new router with all dependencies.

```go
func NewRouter(resolver *runner.TaskResolver, skills []*model.Pipeline, registry *Registry) *Router
```

### NewShellHistory

NewShellHistory loads or creates a shell history file.

```go
func NewShellHistory() *ShellHistory
```

### ParseCorrection

ParseCorrection detects "if I say X, run Y" style corrections.
Returns (phrase, task, true) if matched.

```go
func ParseCorrection(input string) (string, string, bool)
```

### UsageText

UsageText returns the usage help text for non-interactive mode.

```go
func UsageText() string
```

### Exec

Exec processes a single prompt non-interactively and exits.

```go
func (*Agent) Exec(ctx context.Context, prompt, version string) error
```

### Options

Options returns the agent options.

```go
func (*Agent) Options() *Options
```

### Pipelines

Pipelines returns the loaded pipelines.

```go
func (*Agent) Pipelines() []*model.Pipeline
```

### Resolver

Resolver returns the task resolver.

```go
func (*Agent) Resolver() *runner.TaskResolver
```

### Run

Run starts the interactive REPL.

```go
func (*Agent) Run(ctx context.Context, version string) error
```

### WorkDir

WorkDir returns the working directory.

```go
func (*Agent) WorkDir() string
```

### Add

Add records a new alias mapping.

```go
func (*AliasStore) Add(phrase, task string)
```

### Match

Match checks if the input matches any alias phrase.
Returns the target task name, or empty string if no match.

```go
func (*AliasStore) Match(input string) string
```

### AttemptFix

AttemptFix runs the fix task and reports success.

```go
func (*AutoFixer) AttemptFix(ctx context.Context, task *model.ResolvedTask, allPipelines []*model.Pipeline) error
```

### CanFix

CanFix checks if a fix skill exists for the given task.
e.g., if "go:test" fails, checks for "go:fix".

```go
func (*AutoFixer) CanFix(task *model.ResolvedTask) bool
```

### GetFixTask

GetFixTask returns the fix task for a given skill.
"go:test" -> "go:fix", "docker:build" -> "docker:fix".

```go
func (*AutoFixer) GetFixTask(task *model.ResolvedTask) (*model.ResolvedTask, error)
```

### Clear

Clear resets the breadcrumb.

```go
func (*Breadcrumb) Clear()
```

### LastSegment

LastSegment returns the last segment or empty string.

```go
func (*Breadcrumb) LastSegment() string
```

### Pop

Pop removes the last segment.

```go
func (*Breadcrumb) Pop()
```

### Push

Push adds a segment to the breadcrumb trail.
e.g., "go" -> "go > test" -> "go > test > step 1".

```go
func (*Breadcrumb) Push(segment string)
```

### SetStatus

SetStatus updates the current status suffix.
e.g., "running...", "passed", "failed".

```go
func (*Breadcrumb) SetStatus(status string)
```

### String

String renders the breadcrumb as a one-liner.
Output: "go > test > step 1 [running...]".

```go
func (*Breadcrumb) String() string
```

### LearnGreeting

LearnGreeting parses input like "ciao is a greeting" and adds the word.
Returns true if a new greeting was learned.

```go
func (*Greeter) LearnGreeting(input string) (string, bool)
```

### Match

Match checks if the input is a greeting and returns a random response.
Returns empty string if not a greeting.

```go
func (*Greeter) Match(input string) string
```

### Aliases

Aliases returns the alias store for external use.

```go
func (*Parser) Aliases() *AliasStore
```

### AvailableSkills

AvailableSkills returns a list of available skill names for completion.

```go
func (*Parser) AvailableSkills() []string
```

### FindMatches

FindMatches returns all skills matching the input keywords.
Includes description matching.

```go
func (*Parser) FindMatches(keywords []string) []string
```

### Parse

Parse analyzes input and returns an Intent.

```go
func (*Parser) Parse(input string) (*Intent, error)
```

### Get

Get retrieves a command by name or alias.

```go
func (*Registry) Get(name string) *SlashCommand
```

### HelpText

HelpText returns formatted help text for all commands.

```go
func (*Registry) HelpText() string
```

### Register

Register adds a slash command.

```go
func (*Registry) Register(cmd *SlashCommand)
```

### Aliases

Aliases returns the alias store.

```go
func (*Router) Aliases() *AliasStore
```

### AvailableSkills

AvailableSkills returns a list of available skill names.

```go
func (*Router) AvailableSkills() []string
```

### FindMatches

FindMatches returns all skills matching the input keywords.

```go
func (*Router) FindMatches(keywords []string) []string
```

### Greeter

Greeter returns the greeter.

```go
func (*Router) Greeter() *Greeter
```

### LastCommand

LastCommand returns the last command input.

```go
func (*Router) LastCommand() string
```

### Route

Route processes user input following the structure.d2 flow:
1. Is alias? → Replace with alias
2. Semantic parsing
3. Match prompt (skills & targets)?
- single match → execute
- multiple matches → ambiguous
- shell expression → shell exec
- greeting → greeting response
- store correction → save alias
- none → failure

```go
func (*Router) Route(input string) *Route
```

### SetLastCommand

SetLastCommand records the last command for retry functionality.

```go
func (*Router) SetLastCommand(input string, failed bool)
```

### ShellHistory

ShellHistory returns the shell history.

```go
func (*Router) ShellHistory() *ShellHistory
```

### Add

Add records a command execution.

```go
func (*ShellHistory) Add(command string, exitCode int, duration time.Duration, dir string)
```

### FindExact

FindExact returns the most recent entry matching the exact command, or nil.

```go
func (*ShellHistory) FindExact(command string) *ShellHistoryEntry
```

### Match

Match returns shell history entries where the command starts with or
contains the given input. Results are returned most-recent-first.

```go
func (*ShellHistory) Match(input string) []ShellHistoryEntry
```

### Init

Init implements tea.Model.

```go
func (Model) Init() tea.Cmd
```

### Update

Update implements tea.Model.

```go
func (Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

### View

View implements tea.Model.

```go
func (Model) View() tea.View
```
