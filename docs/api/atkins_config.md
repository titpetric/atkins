# Package ./config

```go
import (
	"github.com/titpetric/atkins/config"
)
```

Package `config` owns the atkins configuration document.

Configuration has two layers. The document (.atkins/config.yml,
falling back to the embedded RuntimeDefaultConfig) is the source of
truth. Environment variables are an overlay on top of it, for
containers and CI, and only where they are actually set: an unset or
empty ATKINS_* variable leaves the configured value alone.

Load checks every field, and an empty one takes the default from the
embedded document. A half-written configuration fails at start-up
with a message naming the field, long before the first request.

## Types

<details>
<summary><code>type Config</code></summary>

```go
// Config is the .atkins/config.yml document.
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
// per job and per command to, and what is kept in it.
type StorageConfig struct {
	// Database is the file to record into. Empty discovers it: the
	// atkins.db of the nearest .atkins folder, and
	// $HOME/.atkins/atkins.db for a directory that belongs to no
	// project.
	Database string `yaml:"database" json:"database" env:"ATKINS_DB"`

	// Record writes a row per job and per command of every run. Off
	// records nothing, the same as `--local`, and leaves the database
	// alone.
	//
	// How long a run is kept is not a setting: storage.Retention is 31
	// days on every machine, applied by the run that records.
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
// RuntimeDefaultConfig is the default configuration document, embedded
// so an unconfigured install still has a complete and valid set of
// values and no scatter of zero values.
//
// The configuration menu also writes it when a project has no
// .atkins/config.yml yet, comments and all, so the file a user first
// opens explains itself.
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
- `func (*Menu) SetIO (in io.Reader, out io.Writer)`
- `func (Field) Display () string`
- `func (Field) Kind () string`
- `func (Field) Set (raw string) error`
- `func (Field) String () string`

### `Default`

`Default` returns the embedded configuration.

Every other layer applies on top of this base, and it supplies the
value for any field left empty.

```go
func Default() (*Config, error)
```

### `Discover`

`Discover` walks up from dir looking for .atkins/config.yml, so a
command run deep inside a project still finds the project's
configuration.

```go
func Discover(dir string) string
```

### `EnvironmentNames`

`EnvironmentNames` returns the variables that can override a field,
keyed by the field path they apply to. The configuration menu uses it
to warn that an edit will be overridden at runtime.

```go
func EnvironmentNames() map[string]string
```

### `Load`

`Load` reads the effective configuration for a directory.

The layers, each overriding the last: the embedded default, the user
document at $HOME/.atkins/config.yml, the project document found by
walking up from dir, and finally the environment overlay. Validation
of the result fills anything still empty.

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

`Paths` returns the configuration documents that apply to dir, in the
order Load merges them.

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

`ApplyEnvironment` overlays ATKINS_* variables onto the configuration.

`env` struct tags drive the overlay, so adding a field to the
document adds an override for free. The overlay applies only variables
that hold a non-empty value: an empty variable means "not configured
here", so a container can set one field without wiping the rest of
the document.

The overlay skips a value that doesn't parse, and that is never fatal.
The alternative is a container that fails to start because of a typo in
an optional tuning knob, and the validation pass still catches anything
that would leave the configuration unusable.

```go
func (*Config) ApplyEnvironment(environ []string)
```

### `Encode`

`Encode` renders the configuration as YAML.

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

`Save` writes the configuration, creating the .atkins directory.

```go
func (*Config) Save(path string) error
```

### `Validate`

`Validate` fills empty fields from the embedded defaults and checks what
remains.

The document is two fields wide, and neither can be wrong in a way
worth refusing to start over: a database path that does not open is
reported by the run that tried, and recording is a boolean. What is
left to check is the version, because a document from a newer build
may hold fields this one would silently ignore.

```go
func (*Config) Validate() error
```

### `Run`

`Run` drives the menu until the user quits.

```go
func (*Menu) Run() error
```

### `SetIO`

`SetIO` redirects the menu, so something other than a terminal can
drive it.

```go
func (*Menu) SetIO(in io.Reader, out io.Writer)
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
