# Package ./version

```go
import (
	"github.com/titpetric/atkins/version"
}
```

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

## Consts

<details>
<summary><code>const Name</code></summary>

```go
// Name is the command title.
const Name = "Show version/build information"
```

</details>

## Function symbols

- `func NewCommand (info Info) *cli.Command`
- `func Run (info Info) error`

### NewCommand

NewCommand creates a new version command with build information.

```go
func NewCommand(info Info) *cli.Command
```

### Run

Run will print version information for the build.

```go
func Run(info Info) error
```
