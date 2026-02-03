# Package atkins

```go
import (
	"github.com/titpetric/atkins"
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

- `func Pipeline () *cli.Command`

### Pipeline

Pipeline provides a cli.Command that runs the atkins command pipeline.

```go
func Pipeline () *cli.Command
```


