# Package ./agent/model

```go
import (
	"github.com/titpetric/atkins/agent/model"
}
```

## Types

```go
// Cmd represents a command to be executed by the runtime.
// This is an interface to decouple from specific implementations (e.g., bubbletea).
type Cmd = func() any
```

```go
// Intent represents a parsed user intent.
type Intent struct {
	Type     IntentType
	Raw      string   // Original input
	Keywords []string // Extracted keywords
	Task     string   // Resolved task name (e.g., "go:test")
	Command  string   // Slash command name (without /)
	Args     string   // Arguments for slash command
	Resolved any      // Resolved task reference (type-agnostic)
}
```

```go
// IntentType categorizes user input.
type IntentType int
```

```go
// Model represents the application model interface.
// Implementations provide the actual state and behavior.
type Model interface {
	// Empty interface - implementations define their own methods
}
```

## Consts

```go
// IntentType constants for the enum.
const (
	IntentUnknown IntentType = iota
	IntentTask               // Run a skill/task.
	IntentSlash              // Slash command.
	IntentHelp               // Help request.
	IntentQuit               // Exit request.
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

- `func ExpandKeywords (keywords []string) []string`
- `func StripFillerWords (input string) string`

### ExpandKeywords

ExpandKeywords returns the original keywords plus singularized variants.

```go
func ExpandKeywords(keywords []string) []string
```

### StripFillerWords

```go
func StripFillerWords(input string) string
```
