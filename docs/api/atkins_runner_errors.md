# Package ./runner/errors

```go
import (
	"github.com/titpetric/atkins/runner/errors"
}
```

## Types

<details>
<summary><code>type NoDefaultJobError</code></summary>

```go
// NoDefaultJobError is returned when no default job is found.
type NoDefaultJobError struct {
	Jobs map[string]*model.Job
}
```

</details>

## Function symbols

- `func (*NoDefaultJobError) Error () string`

### Error

Error returns the error hinting a default job should be defined.

```go
func (*NoDefaultJobError) Error() string
```
