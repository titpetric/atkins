# Package ./version

```go
import (
	"github.com/titpetric/atkins/version"
)
```

Package `version` reports what build of atkins is running: the values
the release build injects with -ldflags, and the module and settings
the Go toolchain embeds in the binary.

## Types

<details>
<summary><code>type Info</code></summary>

```go
// Info contains injected build environment information.
type Info struct {
	Version    string
	Commit     string
	CommitTime string
	Branch     string
}
```

</details>

## Function symbols

- `func Run (w io.Writer, info Info) error`

### `Run`

`Run` writes the injected build values and those of the running binary to w.

```go
func Run(w io.Writer, info Info) error
```
