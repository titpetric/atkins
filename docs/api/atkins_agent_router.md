# Package ./agent/router

```go
import (
	"github.com/titpetric/atkins/agent/router"
}
```

## Types

```go
// CommandLookup provides command existence checking.
type CommandLookup interface {
	HasCommand(name string) bool
}
```

```go
// Route represents a routing decision.
type Route struct {
	Type        RouteType
	Raw         string                      // Original input
	Task        string                      // Task name for RouteTask
	Resolved    *model.ResolvedTask         // Resolved task for RouteTask
	Command     string                      // Slash command name for RouteSlash
	Args        string                      // Arguments for RouteSlash
	ShellCmd    string                      // Shell command for RouteShell
	Greeting    string                      // Greeting response for RouteGreeting
	Fortune     string                      // Fortune text for RouteFortune
	Phrase      string                      // Phrase for RouteCorrection
	AliasTask   string                      // Alias target for RouteCorrection
	Ambiguous   bool                        // Multiple matches found
	Matches     []string                    // Matching skills when ambiguous
	HistMatches []history.ShellHistoryEntry // Shell history matches

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
	resolver *runner.TaskResolver
	skills   []*model.Pipeline

	commands CommandLookup
	greeter  *greeting.Greeter
	aliases  *aliases.Aliases

	shellHistory *history.ShellHistory

	// Context for retry/again
	lastInput  string // Last input for retry
	lastFailed bool   // Whether last command failed
}
```

## Consts

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

## Function symbols

- `func NewRouter (resolver *runner.TaskResolver, skills []*model.Pipeline, commands CommandLookup) *Router`
- `func (*Router) Aliases () *aliases.Aliases`
- `func (*Router) AvailableSkills () []string`
- `func (*Router) FindMatches (keywords []string) []string`
- `func (*Router) Greeter () *greeting.Greeter`
- `func (*Router) LastCommand () string`
- `func (*Router) Route (input string) *Route`
- `func (*Router) SetLastCommand (input string, failed bool)`
- `func (*Router) ShellHistory () *history.ShellHistory`

### NewRouter

NewRouter creates a new router with all dependencies.

```go
func NewRouter(resolver *runner.TaskResolver, skills []*model.Pipeline, commands CommandLookup) *Router
```

### Aliases

Aliases returns the alias store.

```go
func (*Router) Aliases() *aliases.Aliases
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
func (*Router) Greeter() *greeting.Greeter
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
- shell expression → shell exec (requires $ prefix)
- greeting → greeting response
- store correction → save alias
- none → failure.

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
func (*Router) ShellHistory() *history.ShellHistory
```
