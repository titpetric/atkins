# Package ./storage

```go
import (
	"github.com/titpetric/atkins/storage"
)
```

Package `storage` owns the atkins event database.

Every run leaves two kinds of row behind: a `run` row per node of the
tree atkins walked, carrying the time that node took with everything
under it, and a `command` row per command a step executed. The first
answers "how long does the release job take", the second "how much of
my day goes to go test".

# Files

A run records to two databases, and Discover names them. The
project's, atkins.db in the nearest .atkins folder, says what this
checkout costs. The workspace's, $HOME/.atkins/atkins.db or the first
file the database setting or ATKINS_DB names, says what every checkout
on the machine costs between them, a question no single project can
answer. A directory with no .atkins folder of its own, or one whose
nearest folder belongs to the workspace, records to the workspace's
alone, once.

The other files the setting names, a list in the form of PATH, are
mirrors of the workspace's: each takes every row the workspace's takes,
and no report reads one. A runner that sets a $HOME per checkout
names the checkout's file first and its own second, and keeps one
history across them. atkins reports a mirror that fails to open once,
and the rest take the row.

Both files hold the same rows. A report of the workspace answers the
questions of a project report: which command, how often, how long. A
database that opens stays in use when the other fails: atkins keeps
the half of the history it can reach, and the report names the file
each half comes from. A failed write to one database leaves the other one's row in
place, and the caller sees the first error.

# One surface

One type, Storage, owns a database. Every method takes a context and
one request value and returns one result and an error. A reader, the
--stats report today and task-ui later, has one surface to learn.
Queries go through github.com/titpetric/pdo, whose generic methods
scan straight into the mig generated types in ../model. A *pdo.PDO
serves one request at a time, and every method allocates one over the
shared pool.

A file opens with a busy timeout and the write-ahead log: two atkins
runs in one checkout write at the same moment, and the second waits
for the first to finish. The driver writes a time in a layout strftime
reads, and the usage buckets group by it; time.Time's own String form
leaves every bucket empty.

# Recording

Record writes the rows of one run in one transaction. A report reads
a run as a whole, the job's time against its steps', and a half
written run would pass for a complete one.

A job row takes the canonical name out of the node id, so `jobs.release`
groups as release from one month to the next while the author edits
its description. A step row keeps the command it ran, the name a reader
of one run looks for. Durations round to the millisecond: truncation
drops half a millisecond per row, and a report summing thousands of
rows comes up seconds short.

# Retention

Retention, thirty-one days, bounds the history. A month of work fits
in it, this week compares against the last, and a checkout nobody
prunes stops growing. Every machine uses the same period. The Recorder purges after each write: the command that
records has the database open anyway, and atkins stays a command with
no service beside it. Purge deletes in bounded batches and leaves the
rest for the next pass, because one DELETE over a year of commands
holds the write lock while every other atkins run on the machine
waits on it.

# Command keys

Normalize reduces a command to the form a row stores: the tool and its
subcommands, then the flags with their values removed, sorted.

```go
go test -v ./...                      go test -v
go test ./... -v                      go test -v
go test -cover -coverprofile=c.out    go test -cover -coverprofile
go test --cpuprofile cpu.out          go test --cpuprofile
CGO_ENABLED=0 go build -o bin/x .     go build -o
sleep 1                               sleep
head -20 file.txt                     head
git rev-parse HEAD | cut -c1-8        git rev-parse HEAD
```

Sorting makes `go test -v -cover` and `go test -cover -v` one row.
Operands drop out, and a flag's argument drops with them with no table
of the flags that take one: `--cpuprofile cpu.out` keeps the flag and
the file reads as an operand. A flag made of digits goes too, so
`head -20` and `head -50` count together.

Key is the name the reports group by: the words of a command up to the
first flag, at most three, none a path or a value. A word holding `/`,
`.`, `=`, `$`, a quote or a number ends it.

```go
go test -v ./...        go test
docker compose up -d    docker compose up
git clone --depth 1 x   git clone
```

A normalized command has the same key as the command it came from. The
reports group rows by key, and the stored form needs no migration. A shell
builtin, `set -eu` at the top of a script among them, never reaches a
row or a report.

## Types

<details>
<summary><code>type Bucket</code></summary>

```go
// Bucket is how wide a usage bucket is.
type Bucket string
```

</details>

<details>
<summary><code>type CommandFilter</code></summary>

```go
// CommandFilter narrows a listing of command rows.
type CommandFilter struct {
	// Invocation keeps the commands of one invocation, and Parent those
	// of one run row.
	Invocation string
	Parent     string

	// Tool and Normalized keep the commands of one tool, or one
	// normalized form.
	Tool       string
	Normalized string

	// Since and Until bound when a command started.
	Since time.Time
	Until time.Time

	// Limit caps the rows. Zero means fifty.
	Limit int
}
```

</details>

<details>
<summary><code>type CommandStat</code></summary>

```go
// CommandStat is one command key and what it has cost, a row of the
// command report.
type CommandStat struct {
	// Command is the key the rows group by, as Key names it: `go test`
	// counts every test run whatever its flags and packages.
	Command string

	// Tool is the first word of it.
	Tool string

	// Runs is how many times it ran.
	Runs int64

	// TotalMs is the time it has taken altogether, the order of the
	// report: a command that takes a second and runs four hundred times
	// costs more than a build that takes a minute.
	TotalMs int64

	// AvgMs and MaxMs are the average and the worst single run.
	AvgMs int64
	MaxMs int64

	// Failures is how many of the runs exited non-zero.
	Failures int64

	// LastAt is when it last ran.
	LastAt *time.Time
}
```

</details>

<details>
<summary><code>type CommandUsage</code></summary>

```go
// CommandUsage is one command key in one bucket of time: the point a
// chart stacked by command draws.
type CommandUsage struct {
	// Bucket is the start of the window, as the database spells a time.
	Bucket string

	// Command is the key, as Key names it, and Tool its first word.
	Command string
	Tool    string

	// Runs and TotalMs are how often it ran in the window and for how
	// long altogether.
	Runs    int64
	TotalMs int64
}
```

</details>

<details>
<summary><code>type Paths</code></summary>

```go
// Paths are the event databases a run records to, the project's and the
// workspace's, and the mirrors that take the same rows. The package
// documentation says why a run writes both.
type Paths struct {
	// Project is the atkins.db of the nearest .atkins folder, empty
	// when the directory belongs to no project, and empty when the
	// nearest folder is the workspace's own.
	Project string

	// Workspace is $HOME/.atkins/atkins.db, or the first file the
	// database setting or $ATKINS_DB names, which every run records to.
	Workspace string

	// Mirrors are the other files the setting names, written as the
	// workspace's is and never read: a runner keeps one for itself
	// beside the one of the checkout.
	Mirrors []string
}
```

</details>

<details>
<summary><code>type PurgeRequest</code></summary>

```go
// PurgeRequest is how much history to drop.
type PurgeRequest struct {
	// Before removes everything that started before this time. A zero
	// time removes nothing.
	Before time.Time

	// Batch bounds one DELETE and Batches how many it runs, so a first
	// pass over a year of history holds the write lock for a moment.
	// Zero means 500 rows and 20 batches, and the next pass takes the
	// rows beyond them.
	Batch   int
	Batches int
}
```

</details>

<details>
<summary><code>type PurgeResult</code></summary>

```go
// PurgeResult counts the rows of each kind a Purge removed.
type PurgeResult struct {
	Runs     int64
	Commands int64
}
```

</details>

<details>
<summary><code>type RecordRequest</code></summary>

```go
// RecordRequest is a finished run to store.
type RecordRequest struct {
	// Log is the event log document of the finished run: its metadata,
	// the tree it walked, every event it collected and the summary.
	Log *eventlog.Log

	// Dir is the directory atkins ran in. Empty records none.
	Dir string
}
```

</details>

<details>
<summary><code>type RecordResult</code></summary>

```go
// RecordResult counts what one Record wrote.
type RecordResult struct {
	// InvocationID is the run row of the pipeline node, the value every
	// row of this invocation carries.
	InvocationID string

	// Runs and Commands count the rows of each kind.
	Runs     int
	Commands int
}
```

</details>

<details>
<summary><code>type Recorder</code></summary>

```go
// Recorder stores finished runs into one Storage and purges what falls
// outside Retention. It implements runner.Recorder, the one view the
// runner has of this package: it passes an event log document and never
// names a table or a column. A nil Recorder records nothing.
type Recorder struct {
	storage *Storage
	dir     string
}
```

</details>

<details>
<summary><code>type Recorders</code></summary>

```go
// Recorders writes a run to each database, the project's and the
// workspace's, the same rows in both. A failed write leaves the others
// to go ahead.
type Recorders []*Recorder
```

</details>

<details>
<summary><code>type RunFilter</code></summary>

```go
// RunFilter narrows a listing of run rows. A zero value lists the most
// recent runs of every kind.
type RunFilter struct {
	// Invocation keeps the rows of one invocation, the tree of one
	// atkins process.
	Invocation string

	// Kind keeps the rows of one kind: pipeline, job or step.
	Kind string

	// Name keeps the rows of one job or step name.
	Name string

	// Since and Until bound when a run started.
	Since time.Time
	Until time.Time

	// Limit caps the rows. Zero means fifty.
	Limit int
}
```

</details>

<details>
<summary><code>type Sample</code></summary>

```go
// Sample is one recorded command: when it started, its key and how it
// ended.
//
// It carries the command's key and never the command line: a command
// line holds its values, and a value can be a token.
type Sample struct {
	// At is when the command started.
	At time.Time

	// Command is the key, as Key names it, and Tool its first word.
	Command string
	Tool    string

	// Kind is where the command came from: a step, a $() substitution
	// or a variable.
	Kind string

	// DurationMs is how long it took and ExitCode how it ended.
	DurationMs int64
	ExitCode   int64

	// Pipeline and Job are what ran it, and InvocationID the atkins
	// process, so the commands of one run group together.
	Pipeline     string
	Job          string
	InvocationID string
}
```

</details>

<details>
<summary><code>type SampleRequest</code></summary>

```go
// SampleRequest selects the commands one by one, each its own row. A
// zero value returns every command the database holds.
type SampleRequest struct {
	// Since and Until bound when a command started, Tool keeps one
	// tool and Dir one directory and below, as for StatsRequest.
	Since time.Time
	Until time.Time
	Tool  string
	Dir   string

	// Limit keeps the newest commands of the window. Zero keeps them
	// all: a reader that buckets the samples needs the whole window,
	// and the retention window bounds it.
	Limit int
}
```

</details>

<details>
<summary><code>type StatsRequest</code></summary>

```go
// StatsRequest narrows the statistics. A zero value reports every
// command the database holds, the slowest first.
type StatsRequest struct {
	// Since and Until bound when a command started. A zero time is no
	// bound at that end.
	Since time.Time
	Until time.Time

	// Tool keeps the commands of one tool, as Normalize names it.
	Tool string

	// Dir keeps the commands that ran in a directory or below it.
	Dir string

	// Limit caps the rows. Zero means twenty, which is a screen.
	Limit int
}
```

</details>

<details>
<summary><code>type Storage</code></summary>

```go
// Storage is one atkins event database: it applies the migrations on
// open, writes the rows of finished runs and answers the reports. Every
// method takes a context and one request value. A nil Storage records
// nothing and closes nothing, so a caller that failed to open one has no
// branch to write.
type Storage struct {
	db   *sqlx.DB
	path string
}
```

</details>

<details>
<summary><code>type ToolStat</code></summary>

```go
// ToolStat is one tool and what it has cost across every form of it, a
// row of the tool report. `go` is one tool whether it builds, tests or
// vets, and its share of a week is the week's share of Go.
type ToolStat struct {
	// Tool is the first word of the command key.
	Tool string

	// Commands is how many distinct command keys of it ran.
	Commands int64

	// Runs is how many times any of them ran.
	Runs int64

	// TotalMs is the time they have taken altogether.
	TotalMs int64

	// Failures is how many of the runs exited non-zero.
	Failures int64
}
```

</details>

<details>
<summary><code>type UsageEntry</code></summary>

```go
// UsageEntry is one tool in one bucket of time, a point of a usage chart.
type UsageEntry struct {
	// Bucket is the start of the window, as the database spells a time.
	Bucket string `db:"bucket"`

	// Tool is the tool the point counts.
	Tool string `db:"tool"`

	// Runs and TotalMs are how often it ran in the window and for how
	// long altogether.
	Runs    int64 `db:"runs"`
	TotalMs int64 `db:"total_ms"`
}
```

</details>

<details>
<summary><code>type UsageRequest</code></summary>

```go
// UsageRequest selects usage over time, a series per tool.
type UsageRequest struct {
	// Since and Until bound when a command started.
	Since time.Time
	Until time.Time

	// Bucket is how wide one point is. Empty means a day.
	Bucket Bucket

	// Tool keeps the commands of one tool. Empty reports every tool.
	Tool string

	// Dir keeps the commands that ran in a directory or below it.
	Dir string
}
```

</details>

## Consts

<details>
<summary><code>const EnvDB</code></summary>

```go
// EnvDB names the workspace's database in place of $HOME/.atkins/atkins.db,
// a list in the form of PATH: the first file is the workspace's, the rest
// are its mirrors.
const EnvDB = "ATKINS_DB"
```

</details>

<details>
<summary><code>const Memory</code></summary>

```go
// Memory is the DSN of a private in-memory database, for tests and for
// a run that must record nowhere.
const Memory = ":memory:"
```

</details>

<details>
<summary><code>const Project</code></summary>

```go
// Project is the migration project name recorded in the migrations
// table.
const Project = "atkins"
```

</details>

<details>
<summary><code>const Retention</code></summary>

```go
// Retention is how long the database holds a run, from its start:
// thirty-one days, fixed. The package documentation says why.
const Retention = 31 * 24 * time.Hour
```

</details>

<details>
<summary><code>const BucketHour, BucketDay, BucketWeek</code></summary>

```go
// The bucket widths of a usage report.
const (
	BucketHour Bucket = "hour"
	BucketDay  Bucket = "day"
	BucketWeek Bucket = "week"
)
```

</details>

## Vars

<details>
<summary><code>var ErrNotFound</code></summary>

```go
// ErrNotFound reports a row that is not there.
var ErrNotFound = errors.New("not found")
```

</details>

## Function symbols

- `func Databases (list string) []string`
- `func Discover (dir string, list string) (Paths, error)`
- `func IsBuiltin (tool string) bool`
- `func Key (command string) string`
- `func NewRecorders (stores []*Storage) Recorders`
- `func NewStorage (list string) ([]*Storage, error)`
- `func NewStorageAt (ctx context.Context, path string) (*Storage, error)`
- `func Normalize (command string) string`
- `func OpenAll (ctx context.Context, paths ...string) ([]*Storage, error)`
- `func Size (path string) int64`
- `func (*Recorder) Record (ctx context.Context, log *eventlog.Log) error`
- `func (*Storage) Close () error`
- `func (*Storage) CommandUsage (ctx context.Context, req UsageRequest) ([]CommandUsage, error)`
- `func (*Storage) Commands (ctx context.Context, filter CommandFilter) ([]model.Command, error)`
- `func (*Storage) Count (ctx context.Context, before time.Time) (*PurgeResult, error)`
- `func (*Storage) Path () string`
- `func (*Storage) Prune (ctx context.Context, before time.Time) (*PurgeResult, error)`
- `func (*Storage) Purge (ctx context.Context, req PurgeRequest) (*PurgeResult, error)`
- `func (*Storage) RawStartedAt (ctx context.Context) (string, error)`
- `func (*Storage) Record (ctx context.Context, req RecordRequest) (*RecordResult, error)`
- `func (*Storage) Recorder () *Recorder`
- `func (*Storage) Run (ctx context.Context, id string) (*model.Run, error)`
- `func (*Storage) Runs (ctx context.Context, filter RunFilter) ([]model.Run, error)`
- `func (*Storage) Samples (ctx context.Context, req SampleRequest) ([]Sample, error)`
- `func (*Storage) Stats (ctx context.Context, req StatsRequest) ([]CommandStat, error)`
- `func (*Storage) Tools (ctx context.Context, req StatsRequest) ([]ToolStat, error)`
- `func (*Storage) Usage (ctx context.Context, req UsageRequest) ([]UsageEntry, error)`
- `func (Paths) All () []string`
- `func (Paths) List () []string`
- `func (Recorders) Record (ctx context.Context, log *eventlog.Log) error`

### `Databases`

`Databases` splits a database setting into its files, dropping empty entries.

```go
func Databases(list string) []string
```

### `Discover`

`Discover` returns the databases for dir and creates their folders. The
workspace's and its mirrors are the files list names, $ATKINS_DB when
list is empty, and $HOME/.atkins/atkins.db when neither names one. The
project's is the atkins.db of the nearest .atkins folder above dir, and
empty when that is one of the files named or there is none.

```go
func Discover(dir string, list string) (Paths, error)
```

### `IsBuiltin`

`IsBuiltin` reports whether a tool is a shell builtin, which the
statistics neither record nor report.

```go
func IsBuiltin(tool string) bool
```

### `Key`

`Key` returns the report key of a command and its tool, or two empty
strings for a builtin or a command with no tool.

```go
func Key(command string) string
```

### `NewRecorders`

`NewRecorders` returns a recorder per storage.

```go
func NewRecorders(stores []*Storage) Recorders
```

### `NewStorage`

`NewStorage` returns the databases a run in the working directory records
to, as Discover names them for list and OpenAll opens them.

```go
func NewStorage(list string) ([]*Storage, error)
```

### `NewStorageAt`

`NewStorageAt` returns the migrated database at path, Memory for a private
one, creating the directory of a file.

```go
func NewStorageAt(ctx context.Context, path string) (*Storage, error)
```

### `Normalize`

`Normalize` returns two empty strings for a command with no tool, else
its stored form and tool, as the package documentation lists them.

```go
func Normalize(command string) string
```

### `OpenAll`

`OpenAll` returns the databases that opened, in order, and the first error.

```go
func OpenAll(ctx context.Context, paths ...string) ([]*Storage, error)
```

### `Size`

`Size` returns the bytes a database takes with its wal and shm files, zero when absent.

```go
func Size(path string) int64
```

### `Record`

`Record` returns nil on a nil Recorder, the write error, or the error
of purging the history older than Retention.

```go
func (*Recorder) Record(ctx context.Context, log *eventlog.Log) error
```

### `Close`

`Close` releases the connection pool; a nil Storage returns nil.

```go
func (*Storage) Close() error
```

### `CommandUsage`

`CommandUsage` reports runs and time per command key per bucket, oldest
first and the most expensive first within a bucket.

```go
func (*Storage) CommandUsage(ctx context.Context, req UsageRequest) ([]CommandUsage, error)
```

### `Commands`

`Commands` lists command rows, newest first.

```go
func (*Storage) Commands(ctx context.Context, filter CommandFilter) ([]model.Command, error)
```

### `Count`

`Count` returns how many runs and commands started before a time, every
row for a zero time: what a prune would remove, or what remains.

```go
func (*Storage) Count(ctx context.Context, before time.Time) (*PurgeResult, error)
```

### `Path`

`Path` returns the file the storage opened, for a report that names it.

```go
func (*Storage) Path() string
```

### `Prune`

`Prune` removes every run and command started before a time, batch
after batch until none remains, then releases the space: it vacuums
the file and truncates its write-ahead log. A zero time keeps every
row.

```go
func (*Storage) Prune(ctx context.Context, before time.Time) (*PurgeResult, error)
```

### `Purge`

`Purge` returns an empty result for a zero time, an error from a DELETE,
or the rows it removed, in bounded batches, of the history started
before req.Before.

```go
func (*Storage) Purge(ctx context.Context, req PurgeRequest) (*PurgeResult, error)
```

### `RawStartedAt`

`RawStartedAt` returns the started_at of a command row as the database
holds the text, or the error of an empty table.

```go
func (*Storage) RawStartedAt(ctx context.Context) (string, error)
```

### `Record`

`Record` returns an empty result for nothing to record, the error that
rolls the transaction back, the commit error, or what it wrote: a run
row per node of the tree and a command row per command the run executed.

```go
func (*Storage) Record(ctx context.Context, req RecordRequest) (*RecordResult, error)
```

### `Recorder`

`Recorder` returns nil for a nil Storage, else a recorder of the working
directory.

```go
func (*Storage) Recorder() *Recorder
```

### `Run`

`Run` returns one run row by ID, or ErrNotFound.

```go
func (*Storage) Run(ctx context.Context, id string) (*model.Run, error)
```

### `Runs`

`Runs` lists run rows, newest first.

```go
func (*Storage) Runs(ctx context.Context, filter RunFilter) ([]model.Run, error)
```

### `Samples`

`Samples` returns the query error, or the commands of the request one by
one, oldest first, each with the job above its step.

```go
func (*Storage) Samples(ctx context.Context, req SampleRequest) ([]Sample, error)
```

### `Stats`

`Stats` returns the query error, or the command keys by the time they
have taken altogether, the most expensive first.

```go
func (*Storage) Stats(ctx context.Context, req StatsRequest) ([]CommandStat, error)
```

### `Tools`

`Tools` returns the query error, or the command keys rolled up per tool,
the most expensive first.

```go
func (*Storage) Tools(ctx context.Context, req StatsRequest) ([]ToolStat, error)
```

### `Usage`

`Usage` returns the error of an unknown bucket, or the runs and time per
tool per bucket, oldest first: the series of a usage chart.

```go
func (*Storage) Usage(ctx context.Context, req UsageRequest) ([]UsageEntry, error)
```

### `All`

`All` returns the databases a run writes: List and the mirrors after it.

```go
func (Paths) All() []string
```

### `List`

`List` returns the databases a report reads: the workspace's alone
without a project, else both, the project's first.

```go
func (Paths) List() []string
```

### `Record`

`Record` returns the first error after writing to every database.

```go
func (Recorders) Record(ctx context.Context, log *eventlog.Log) error
```
