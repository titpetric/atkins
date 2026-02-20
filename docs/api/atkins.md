# Package atkins

```go
import (
	"github.com/titpetric/atkins"
}
```

## Types

```go
// Options holds pipeline command-line arguments
type Options struct {
	File			string
	Job			string
	List			bool
	Lint			bool
	Debug			bool
	LogFile			string
	FinalOnly		bool
	WorkingDirectory	string
	Jail			bool

	FlagSet	*cli.FlagSet
}
```

## Vars

```go
// Version information injected at build time via ldflags
var (
	Version		= "dev"
	Commit		= "unknown"
	CommitTime	= "unknown"
	Branch		= "unknown"
)
```

## Function symbols

- `func NewOptions () *Options`
- `func Pipeline () *cli.Command`
- `func (*Options) Bind (fs *cli.FlagSet)`

### Pipeline

Pipeline provides a cli.Command that runs the atkins command pipeline.

```go
func Pipeline () *cli.Command
```

### NewOptions

```go
func NewOptions () *Options
```

### Bind

```go
func (*Options) Bind (fs *cli.FlagSet)
```


