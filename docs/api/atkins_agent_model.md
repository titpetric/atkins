# Package ./agent/model

```go
import (
	"github.com/titpetric/atkins/agent/model"
}
```

## Types

<details>
<summary><code>type Cmd</code></summary>

```go
// Cmd represents a command to be executed by the runtime.
// This is an interface to decouple from specific implementations (e.g., bubbletea).
type Cmd = func() any
```

</details>

<details>
<summary><code>type Intent</code></summary>

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

</details>

<details>
<summary><code>type IntentType</code></summary>

```go
// IntentType categorizes user input.
type IntentType int
```

</details>

<details>
<summary><code>type Model</code></summary>

```go
// Model represents the application model interface.
// Implementations provide the actual state and behavior.
type Model interface {
	// Empty interface - implementations define their own methods
}
```

</details>

## Consts

<details>
<summary><code>const IntentUnknown, IntentTask, IntentSlash, IntentHelp, IntentQuit</code></summary>

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

</details>

## Vars

<details>
<summary><code>var FillerWords</code></summary>

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

</details>

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
