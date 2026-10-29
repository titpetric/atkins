# Package ./config

```go
import (
	"github.com/titpetric/atkins/config"
)
```

Package `config` owns the atkins configuration document, .atkins/config.yml.

Configuration comes in layers, each overriding the last: the embedded
default (RuntimeDefaultConfig), the user document at
$HOME/.atkins/config.yml, the project document found by walking up from
the working directory, and the ATKINS_* environment overlay. Load
assembles them; LoadFile reads one document over the default, and skips
the user layer and the environment, for the configuration menu to edit.

The embedded default holds a complete and valid set of values. An
unconfigured install runs on those values. Every field a document
leaves empty keeps the default, because a document unmarshals over the
struct holding the layers below it. The menu writes the embedded
document, comments and all, when a project has none yet. The file a
user first opens explains itself.

The environment overlay serves containers and CI. `env` struct tags
drive it, and a field added to the document gains an override with no
further code. It applies a variable only when the variable holds a
value: an empty ATKINS_* variable means "not configured here". A
container can set one field and keep the rest of the document. It skips
a value that does not parse and carries on, and a container with a typo
in an optional setting still starts.

Validation fills what remains empty and checks the version. The
document is two fields wide: the run that opens the database reports a
path that does not open, and recording is a boolean. A document from a
newer build may hold fields this build ignores, and a newer version
fails validation.

## Types

<details>
<summary><code>type Config</code></summary>

```go
// Config is the .atkins/config.yml document. A value holds the layers
// merged so far; Load and LoadFile return one that validates, and Save
// writes it back.
type Config struct {
	// Version identifies the document format.
	Version int `yaml:"version" json:"version"`

	// Storage is what a run records and where.
	Storage StorageConfig `yaml:"storage" json:"storage"`
}
```

</details>

<details>
<summary><code>type Field</code></summary>

```go
// Field is one editable value in the configuration document.
type Field struct {
	// Path is the dotted document path, e.g. "storage.retention".
	Path string

	// Env is the variable that overrides it at runtime, if any.
	Env string

	// value points at the live field so Set writes through.
	value reflect.Value
}
```

</details>

<details>
<summary><code>type Menu</code></summary>

```go
// Menu edits a configuration document interactively.
type Menu struct {
	// Path is the document the menu edits.
	Path string

	config *Config
	in     *bufio.Reader
	out    io.Writer

	// dirty tracks unsaved edits so quitting can warn.
	dirty bool
}
```

</details>

<details>
<summary><code>type StorageConfig</code></summary>

```go
// StorageConfig is the event database: the file every run appends a row
// per job and per command to, and what it holds.
type StorageConfig struct {
	// Database is the workspace's file in place of
	// $HOME/.atkins/atkins.db, or a list of files in the form of PATH:
	// the first is the workspace's, the rest are mirrors that take the
	// same rows and are never read. The project's, the atkins.db of the
	// nearest .atkins folder, records as well either way.
	Database string `yaml:"database" json:"database" env:"ATKINS_DB"`

	// Record writes a row per job and per command of every run. Off
	// records nothing, the same as `--local`, and leaves the database
	// alone.
	//
	// Retention is no setting: storage.Retention is 31 days on every
	// machine, applied by the run that records.
	Record bool `yaml:"record" json:"record" env:"ATKINS_RECORD"`
}
```

</details>

## Consts

<details>
<summary><code>const Dir</code></summary>

```go
// Dir is the directory holding project and user configuration.
const Dir = ".atkins"
```

</details>

<details>
<summary><code>const FileName</code></summary>

```go
// FileName is the configuration document inside an .atkins directory.
const FileName = "config.yml"
```

</details>

<details>
<summary><code>const Version</code></summary>

```go
// Version is the document version this build understands.
const Version = 1
```

</details>

## Vars

<details>
<summary><code>var RuntimeDefaultConfig</code></summary>

```go
// RuntimeDefaultConfig is the default configuration document, the base
// layer under every other.
//
//go:embed config.yml
var RuntimeDefaultConfig []byte
```

</details>

## Function symbols

- `func Default () (*Config, error)`
- `func Discover (dir string) string`
- `func EnvironmentNames () map[string]string`
- `func Load (dir string) (*Config, error)`
- `func LoadFile (path string) (*Config, error)`
- `func NewMenu (path string) (*Menu, error)`
- `func Paths (dir string) []string`
- `func ProjectPath (dir string) string`
- `func SplitList (value string) []string`
- `func (*Config) ApplyEnvironment (environ []string)`
- `func (*Config) Encode () ([]byte, error)`
- `func (*Config) Field (path string) (Field, bool)`
- `func (*Config) Fields () []Field`
- `func (*Config) Save (path string) error`
- `func (*Config) Validate () error`
- `func (*Menu) Run () error`
- `func (*Menu) SetIO (w io.Writer, r io.Reader)`
- `func (Field) Display () string`
- `func (Field) Kind () string`
- `func (Field) Set (raw string) error`
- `func (Field) String () string`

### `Default`

`Default` returns the embedded configuration, or the error parsing it.

```go
func Default() (*Config, error)
```

### `Discover`

`Discover` walks up from dir looking for .atkins/config.yml. A command
run deep inside a project finds the project's configuration.

```go
func Discover(dir string) string
```

### `EnvironmentNames`

`EnvironmentNames` returns the override variable of each field, keyed by field path.

```go
func EnvironmentNames() map[string]string
```

### `Load`

`Load` returns the merged and validated configuration for a directory,
or the error of the default, a document or validation.

```go
func Load(dir string) (*Config, error)
```

### `LoadFile`

`LoadFile` reads a single document, with defaults and validation but
without the user layer or the environment overlay. The
configuration menu edits this result.

```go
func LoadFile(path string) (*Config, error)
```

### `NewMenu`

`NewMenu` opens the document at path for editing, starting from the
embedded defaults when it doesn't exist yet.

```go
func NewMenu(path string) (*Menu, error)
```

### `Paths`

`Paths` returns the documents that apply to dir, in the order Load merges them.

```go
func Paths(dir string) []string
```

### `ProjectPath`

`ProjectPath` returns where a project's configuration belongs: the
document that exists, or the path where a new one belongs.

```go
func ProjectPath(dir string) string
```

### `SplitList`

`SplitList` parses a comma separated value into a list, dropping empty
entries so `a,,b` and `a , b` both return two items.

```go
func SplitList(value string) []string
```

### `ApplyEnvironment`

`ApplyEnvironment` overlays the non-empty ATKINS_* variables of environ.

```go
func (*Config) ApplyEnvironment(environ []string)
```

### `Encode`

`Encode` renders the configuration as YAML indented by two spaces.

```go
func (*Config) Encode() ([]byte, error)
```

### `Field`

`Field` returns one field by path.

```go
func (*Config) Field(path string) (Field, bool)
```

### `Fields`

`Fields` returns every editable field, in document order.

```go
func (*Config) Fields() []Field
```

### `Save`

`Save` writes the configuration to path, owner-only, creating its directory.

```go
func (*Config) Save(path string) error
```

### `Validate`

`Validate` fills empty fields from the embedded defaults, and returns an
error for a version newer than this build.

```go
func (*Config) Validate() error
```

### `Run`

`Run` drives the menu until the user quits.

```go
func (*Menu) Run() error
```

### `SetIO`

`SetIO` points the menu at w and r in place of the terminal.

```go
func (*Menu) SetIO(w io.Writer, r io.Reader)
```

### `Display`

`Display` renders the value for a listing, marking empties so a blank
line is never ambiguous.

```go
func (Field) Display() string
```

### `Kind`

`Kind` names the value type, for the edit prompt.

```go
func (Field) Kind() string
```

### `Set`

`Set` parses and assigns a value, reporting what didn't parse.

```go
func (Field) Set(raw string) error
```

### `String`

`String` renders the current value for display.

```go
func (Field) String() string
```
