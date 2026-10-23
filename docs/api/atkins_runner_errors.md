# Package ./runner/errors

```go
import (
	"github.com/titpetric/atkins/runner/errors"
)
```

Package `errors` holds the error types of the runner that a caller tells
apart with errors.As, such as the NoDefaultJobError a pipeline without a
default job returns.

## Types

<details>
<summary><code>type NoDefaultJobError</code></summary>

```go
// NoDefaultJobError reports a pipeline that has no default job.
type NoDefaultJobError struct {
	Jobs map[string]*model.Job
}
```

</details>

## Function symbols

- `func (*NoDefaultJobError) Error () string`

### `Error`

`Error` returns the error hinting that the pipeline needs a default job.

```go
func (*NoDefaultJobError) Error() string
```
