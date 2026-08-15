# Package ./config

```go
import (
	"github.com/titpetric/atkins/config"
}
```

Package config owns the atkins configuration document.

Configuration has two layers. The document — .atkins/config.yml,
falling back to the embedded RuntimeDefaultConfig — is the source of
truth. Environment variables are an overlay on top of it, for
containers and CI, and only where they are actually set: an unset or
empty ATKINS_* variable leaves the configured value alone.

Every field is checked on load, and an empty one takes the default
from the embedded document. A half-written config fails at start-up
with a message naming the field, rather than at the first request.

## Types

```go
// AgentConfig is the worker that runs jobs.
type AgentConfig struct {
	// ID is the agent's identity in job leases. Empty means the
	// hostname.
	ID string `yaml:"id" json:"id" env:"ATKINS_AGENT_ID"`

	// Token is the enrolment secret, matching the server's
	// agent_token.
	Token string `yaml:"token" json:"token" env:"ATKINS_AGENT_TOKEN"`

	// Labels advertise what this agent can run.
	Labels []string `yaml:"labels" json:"labels" env:"ATKINS_LABELS"`

	DataDir           string        `yaml:"data_dir" json:"data_dir" env:"ATKINS_DATA_DIR"`
	PollInterval      time.Duration `yaml:"poll_interval" json:"poll_interval" env:"ATKINS_POLL_INTERVAL"`
	JobTimeout        time.Duration `yaml:"job_timeout" json:"job_timeout" env:"ATKINS_JOB_TIMEOUT"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" json:"heartbeat_interval" env:"ATKINS_HEARTBEAT_INTERVAL"`
	Shell             string        `yaml:"shell" json:"shell" env:"ATKINS_SHELL"`
	PreferHTTPS       bool          `yaml:"prefer_https" json:"prefer_https" env:"ATKINS_GIT_PREFER_HTTPS"`
}
```

```go
// ClientConfig is the dispatching side.
type ClientConfig struct {
	// Server is the URL runs are dispatched to. Empty falls back to
	// the server this machine last logged in to.
	Server string `yaml:"server" json:"server" env:"ATKINS_SERVER"`

	// Credentials is where login tokens are stored. Empty means
	// $HOME/.atkins/credentials.json.
	Credentials string `yaml:"credentials" json:"credentials" env:"ATKINS_CREDENTIALS"`

	// Labels constrain which agents may run this machine's jobs.
	Labels []string `yaml:"labels" json:"labels" env:"ATKINS_LABELS"`

	// Dispatch hands runs to the server when logged in. Off means
	// always run locally.
	Dispatch bool `yaml:"dispatch" json:"dispatch" env:"ATKINS_DISPATCH"`

	// Timeout bounds an API call before the run falls back to local.
	Timeout time.Duration `yaml:"timeout" json:"timeout" env:"ATKINS_TIMEOUT"`
}
```

```go
// Config is the .atkins/config.yml document.
type Config struct {
	// Version identifies the document format.
	Version int `yaml:"version" json:"version"`

	// Client is how this machine talks to a server.
	Client ClientConfig `yaml:"client" json:"client"`

	// Server configures `atkins server`.
	Server ServerConfig `yaml:"server" json:"server"`

	// Agent configures `atkins worker`.
	Agent AgentConfig `yaml:"agent" json:"agent"`
}
```

```go
// Field is one editable value in the configuration document.
type Field struct {
	// Path is the dotted document path, e.g. "server.lease_ttl".
	Path string

	// Env is the variable that overrides it at runtime, if any.
	Env string

	// Secret marks a value that should not be echoed back in full.
	Secret bool

	// value points at the live field so Set writes through.
	value reflect.Value
}
```

```go
// Menu edits a configuration document interactively.
type Menu struct {
	// Path is the document being edited.
	Path string

	config *Config
	in     *bufio.Reader
	out    io.Writer

	// dirty tracks unsaved edits so quitting can warn.
	dirty bool
}
```

```go
// ServerConfig is the CI/CD server.
type ServerConfig struct {
	Addr       string `yaml:"addr" json:"addr" env:"PLATFORM_SERVER_ADDR"`
	Database   string `yaml:"database" json:"database" env:"PLATFORM_DB_DEFAULT"`
	Connection string `yaml:"connection" json:"connection" env:"ATKINS_DB_CONNECTION"`

	// SigningKey signs access tokens. Required; there is deliberately
	// no default.
	SigningKey string `yaml:"signing_key" json:"signing_key" env:"ATKINS_SIGNING_KEY"`

	// AgentToken is the shared secret agents enrol with.
	AgentToken string `yaml:"agent_token" json:"agent_token" env:"ATKINS_AGENT_TOKEN"`

	AllowRegistration bool          `yaml:"allow_registration" json:"allow_registration" env:"ATKINS_ALLOW_REGISTRATION"`
	TokenTTL          time.Duration `yaml:"token_ttl" json:"token_ttl" env:"ATKINS_TOKEN_TTL"`
	SessionTTL        time.Duration `yaml:"session_ttl" json:"session_ttl" env:"ATKINS_SESSION_TTL"`
	MaxJobDepth       int64         `yaml:"max_job_depth" json:"max_job_depth" env:"ATKINS_MAX_JOB_DEPTH"`
	LeaseTTL          time.Duration `yaml:"lease_ttl" json:"lease_ttl" env:"ATKINS_LEASE_TTL"`
	ReclaimInterval   time.Duration `yaml:"reclaim_interval" json:"reclaim_interval" env:"ATKINS_RECLAIM_INTERVAL"`
}
```

## Consts

```go
// Dir is the directory holding project and user configuration.
const Dir = ".atkins"
```

```go
// FileName is the configuration document inside an .atkins directory.
const FileName = "config.yml"
```

```go
// Version is the document version this build understands.
const Version = 1
```

## Vars

```go
// RuntimeDefaultConfig is the default configuration document, embedded
// so an unconfigured install still has a complete and valid set of
// values rather than a scatter of zero values.
//
// It is also what `atkins --config` writes when a project has no
// .atkins/config.yml yet, comments and all, so the file a user first
// opens explains itself.
//
//go:embed config.yml
var RuntimeDefaultConfig []byte
```

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

### Default

Default returns the embedded configuration.

It is the base every other layer is applied to, and the source of the
value substituted for any field left empty.

```go
func Default() (*Config, error)
```

### Discover

Discover walks up from dir looking for .atkins/config.yml, so a
command run deep inside a project still finds the project's
configuration.

```go
func Discover(dir string) string
```

### EnvironmentNames

EnvironmentNames returns the variables that can override a field,
keyed by the field path they apply to. The configuration menu uses it
to warn that an edit will be overridden at runtime.

```go
func EnvironmentNames() map[string]string
```

### Load

Load reads the effective configuration for a directory.

The layers, each overriding the last: the embedded default, the user
document at $HOME/.atkins/config.yml, the project document found by
walking up from dir, and finally the environment overlay. The result
is validated, which fills anything still empty.

```go
func Load(dir string) (*Config, error)
```

### LoadFile

LoadFile reads a single document, with defaults and validation but
without the user layer or the environment overlay. It is what the
configuration menu edits.

```go
func LoadFile(path string) (*Config, error)
```

### NewMenu

NewMenu opens the document at path for editing, starting from the
embedded defaults when it doesn't exist yet.

```go
func NewMenu(path string) (*Menu, error)
```

### Paths

Paths returns the configuration documents that apply to dir, in the
order they should be merged.

```go
func Paths(dir string) []string
```

### ProjectPath

ProjectPath returns where a project's configuration belongs: the
document that exists, or the path one would be created at.

```go
func ProjectPath(dir string) string
```

### SplitList

SplitList parses a comma separated value into a list, dropping empty
entries so `a,,b` and `a , b` both give two items.

```go
func SplitList(value string) []string
```

### ApplyEnvironment

ApplyEnvironment overlays ATKINS_* variables onto the configuration.

The overlay is driven by `env` struct tags, so adding a field to the
document gives it an override for free. Only variables that are set
and non-empty are applied: an empty variable means "not configured
here", which is what lets a container set one field without wiping
the rest of the document.

A value that doesn't parse is ignored rather than fatal. The
alternative is a container that refuses to start because of a typo in
an optional tuning knob, and Validate still catches anything that
would leave the config unusable.

```go
func (*Config) ApplyEnvironment(environ []string)
```

### Encode

Encode renders the configuration as YAML.

```go
func (*Config) Encode() ([]byte, error)
```

### Field

Field returns one field by path.

```go
func (*Config) Field(path string) (Field, bool)
```

### Fields

Fields returns every editable field, in document order.

```go
func (*Config) Fields() []Field
```

### Save

Save writes the configuration, creating the .atkins directory.

```go
func (*Config) Save(path string) error
```

### Validate

Validate fills empty fields from the embedded defaults and checks
what remains.

Defaulting and checking are one pass on purpose: "unset" and
"invalid" are different problems, and only the second is worth
refusing to start over. A missing timeout is a value nobody chose; a
negative one is a value somebody got wrong.

```go
func (*Config) Validate() error
```

### Run

Run drives the menu until the user quits.

```go
func (*Menu) Run() error
```

### SetIO

SetIO redirects the menu, so it can be driven from something other
than a terminal.

```go
func (*Menu) SetIO(in io.Reader, out io.Writer)
```

### Display

Display renders the value for a listing, masking secrets and marking
empties so a blank line is never ambiguous.

```go
func (Field) Display() string
```

### Kind

Kind names the value type, for the edit prompt.

```go
func (Field) Kind() string
```

### Set

Set parses and assigns a value, reporting what didn't parse.

```go
func (Field) Set(raw string) error
```

### String

String renders the current value for display.

```go
func (Field) String() string
```
