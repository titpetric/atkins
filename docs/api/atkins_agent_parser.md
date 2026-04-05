# Package ./agent/parser

```go
import (
	"github.com/titpetric/atkins/agent/parser"
}
```

## Types

```go
type Intent = agentmodel.Intent
```

```go
// Parser parses user input into intents.
type Parser struct {
	resolver *runner.TaskResolver
	skills   []*model.Pipeline
	aliases  *aliases.AliasStore
}
```

## Function symbols

- `func NewParser (resolver *runner.TaskResolver, skills []*model.Pipeline) *Parser`
- `func (*Parser) Aliases () *aliases.AliasStore`
- `func (*Parser) AvailableSkills () []string`
- `func (*Parser) FindMatches (keywords []string) []string`
- `func (*Parser) Parse (input string) (*agentmodel.Intent, error)`

### NewParser

NewParser creates a new intent parser.

```go
func NewParser(resolver *runner.TaskResolver, skills []*model.Pipeline) *Parser
```

### Aliases

Aliases returns the alias store for external use.

```go
func (*Parser) Aliases() *aliases.AliasStore
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

Parse analyzes input and returns an agentmodel.Intent.

```go
func (*Parser) Parse(input string) (*agentmodel.Intent, error)
```
