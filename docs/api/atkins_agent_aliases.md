# Package ./agent/aliases

```go
import (
	"github.com/titpetric/atkins/agent/aliases"
}
```

## Types

```go
// AliasEntry maps a natural language phrase to a prompt.
// The prompt can be a shell command, prompt target, or any input.
type AliasEntry struct {
	Phrase string `yaml:"phrase"`
	Prompt string `yaml:"prompt"`
}
```

```go
// AliasStore manages user-defined phrase to prompt mappings.
type AliasStore struct {
	Aliases []AliasEntry `yaml:"aliases"`

	path string
}
```

```go
// Aliases is the alias store type alias for convenience.
type Aliases = AliasStore
```

## Function symbols

- `func NewAliasStore () *AliasStore`
- `func ParseCorrection (input string) (string, string, bool)`
- `func (*AliasStore) Add (phrase,prompt string)`
- `func (*AliasStore) Match (input string) string`

### NewAliasStore

NewAliasStore loads or creates the alias store.

```go
func NewAliasStore() *AliasStore
```

### ParseCorrection

ParseCorrection detects "if I say X, run Y" style corrections.
Returns (phrase, task, true) if matched.

```go
func ParseCorrection(input string) (string, string, bool)
```

### Add

Add records a new alias mapping.

```go
func (*AliasStore) Add(phrase, prompt string)
```

### Match

Match checks if the input matches any alias phrase.
Returns the target task name, or empty string if no match.

```go
func (*AliasStore) Match(input string) string
```
