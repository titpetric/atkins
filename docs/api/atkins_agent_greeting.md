# Package ./agent/greeting

```go
import (
	"github.com/titpetric/atkins/agent/greeting"
}
```

## Types

```go
// Greeter handles greeting detection and responses.
type Greeter struct {
	groups []GreetingGroup
}
```

```go
// GreetingConfig is the YAML structure for ~/.atkins/greetings.yaml.
type GreetingConfig struct {
	Greetings []GreetingGroup `yaml:"greetings"`
}
```

```go
// GreetingGroup maps a language/group to its trigger words and responses.
type GreetingGroup struct {
	Keywords  []string `yaml:"keywords"`
	Responses []string `yaml:"responses"`
}
```

## Function symbols

- `func Fortune () string`
- `func MatchFortune (input string) bool`
- `func NewGreeter () *Greeter`
- `func (*Greeter) LearnGreeting (input string) (string, bool)`
- `func (*Greeter) Match (input string) string`

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

### NewGreeter

NewGreeter creates a greeter with built-in defaults merged with user config.

```go
func NewGreeter() *Greeter
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
