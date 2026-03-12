# Package ./runner/errors

```go
import (
	"github.com/titpetric/atkins/runner/errors"
}
```

## Types

```go
// NoDefaultJobError is returned when no default job is found.
type NoDefaultJobError struct {
	Jobs map[string]*model.Job
}
```

## Function symbols

- `func (*NoDefaultJobError) Error () string`

### Error

Error returns the error hinting a default job should be defined.

```go
func (*NoDefaultJobError) Error() string
```
