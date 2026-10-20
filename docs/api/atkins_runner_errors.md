# Package ./runner/errors

```go
import (
	"github.com/titpetric/atkins/runner/errors"
)
```

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
