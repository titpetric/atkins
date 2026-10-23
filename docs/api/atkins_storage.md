# Package ./storage

```go
import (
	"github.com/titpetric/atkins/storage"
)
```

Package `storage` owns the atkins event database.

Every run leaves two kinds of row behind: a `run` row per node of the
tree atkins walked, carrying the time that node took with everything
under it, and a `command` row per command a step actually executed.
The first answers "how long does the release job take", the second
"how much of my day is go test".

The file is atkins.db in the nearest .atkins folder, and
$HOME/.atkins/atkins.db when a directory belongs to no project. One
type, Storage, owns all of it: every method takes a context and one
request value and returns one result and an error, so a reader - the
--stats report today, task-ui later - has one surface to learn.

Queries go through github.com/titpetric/pdo, whose generic methods
scan straight into the mig generated types in ../model. A *pdo.PDO is
request scoped and not safe for concurrent use, so every method
allocates one over the shared pool.

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
// CommandStat is one command key and what it has cost.
type CommandStat struct {
	// Command is the key the rows group by, as Key names it: `go test`
	// counts every test run whatever its flags and packages.
	Command string

	// Tool is the first word of it.
	Tool string

	// Runs is how many times it ran.
	Runs int64

	// TotalMs is the time it has taken altogether, which is what the
	// report sorts by: a command that takes a second and runs four
	// hundred times costs more than a build that takes a minute.
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
// Paths are the event databases a run records to.
// A run records to both, because the two answer different questions.
// The project's says what this checkout costs, which is the question
// standing in it; the workspace's says what every checkout on the
// machine costs between them, which is the one a week of work asks and
// no single project can answer.
type Paths struct {
	// Project is the atkins.db of the nearest .atkins folder, empty
	// when the directory belongs to no project, and empty when the
	// nearest folder is the workspace's own.
	Project string

	// Workspace is $HOME/.atkins/atkins.db, which every run records to.
	Workspace string
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
	// pass over a year of history does not hold the write lock for
	// minutes. Zero means 500 rows and 20 batches, and what is left
	// over goes on the next pass.
	Batch   int
	Batches int
}
```

</details>

<details>
<summary><code>type PurgeResult</code></summary>

```go
// PurgeResult says how many rows of each kind went.
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
	// Log is the event log document the run ended with: its metadata,
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
// RecordResult says what one Record wrote.
type RecordResult struct {
	// InvocationID is the run row of the pipeline node, the value every
	// row of this invocation carries.
	InvocationID string

	// Runs and Commands are how many rows of each kind were written.
	Runs     int
	Commands int
}
```

</details>

<details>
<summary><code>type Recorder</code></summary>

```go
// Recorder stores finished runs. It satisfies runner.Recorder, which is
// the only thing the runner knows about this package: the runner hands
// over an event log document and never names a table or a column.
type Recorder struct {
	storage *Storage
	dir     string
}
```

</details>

<details>
<summary><code>type Recorders</code></summary>

```go
// Recorders writes a run to each of them: the project's database and
// the workspace's.
//
// The write is the same rows twice rather than a summary in one and the
// detail in the other, because a report of the workspace asks the same
// questions of it that a report of a project asks: which command, how
// often, how long.
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
// Storage is the atkins event database.
type Storage struct {
	db   *sqlx.DB
	path string
}
```

</details>

<details>
<summary><code>type ToolStat</code></summary>

```go
// ToolStat is one tool and what it has cost, across every form of it.
// It answers the question a per-command report cannot: `go` is one tool
// whether it builds, tests or vets, and the share of a week it takes is
// the share of a week spent on Go.
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
// UsageEntry is one tool in one bucket of time: the point a chart draws.
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
// UsageRequest asks for usage over time, a series per tool.
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
// Retention is how long a recorded run is kept, measured from when it
// started.
//
// Thirty-one days, fixed. It is long enough to hold a month of work and
// compare this week against the last, and short enough that a checkout
// nobody prunes does not grow without bound. A window nobody has to
// choose is a window that is right on every machine, including the ones
// where the configuration was never opened.
const Retention = 31 * 24 * time.Hour
```

</details>

<details>
<summary><code>const BucketHour, BucketDay, BucketWeek</code></summary>

```go
// The buckets a usage report can ask for.
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

- `func Discover (dir string) (Paths, error)`
- `func IsBuiltin (tool string) bool`
- `func Key (command string) string`
- `func NewRecorders (stores []*Storage) Recorders`
- `func NewStorage () ([]*Storage, error)`
- `func NewStorageAt (ctx context.Context, path string) (*Storage, error)`
- `func Normalize (command string) string`
- `func OpenAll (ctx context.Context, paths ...string) ([]*Storage, error)`
- `func (*Recorder) Record (ctx context.Context, log *eventlog.Log) error`
- `func (*Storage) Close () error`
- `func (*Storage) CommandUsage (ctx context.Context, req UsageRequest) ([]CommandUsage, error)`
- `func (*Storage) Commands (ctx context.Context, filter CommandFilter) ([]model.Command, error)`
- `func (*Storage) Path () string`
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
- `func (Paths) List () []string`
- `func (Recorders) Record (ctx context.Context, log *eventlog.Log) error`

### `Discover`

`Discover` returns the event databases for a directory.

It walks up from dir looking for a .atkins folder and takes the
atkins.db inside the first one it finds as the project's, so every
run in a checkout records to the same file. $HOME/.atkins/atkins.db
is the workspace's, and Discover creates the folder.

A directory inside $HOME with no .atkins folder of its own finds the
workspace's, which is then the only database: a run records to it
once rather than twice.

```go
func Discover(dir string) (Paths, error)
```

### `IsBuiltin`

`IsBuiltin` reports whether a tool is a shell builtin, which the
statistics neither record nor report.

```go
func IsBuiltin(tool string) bool
```

### `Key`

`Key` is the name the reports group a command by: its words up to the
first argument that starts with `-`, at most three, none a path or a
value (a word with `/`, `.`, `=`, `$`, a quote or a number ends it):

```go
go test -v ./...        go test
docker compose up -d    docker compose up
git clone --depth 1 x   git clone
```

A normalized command keys the same as the command it came from.

```go
func Key(command string) string
```

### `NewRecorders`

`NewRecorders` returns a recorder per storage, for a run to record to
each.

```go
func NewRecorders(stores []*Storage) Recorders
```

### `NewStorage`

`NewStorage` opens the event databases for the working directory,
applies the migrations and returns them, the project's first.

The paths come from Discover. Opening is the only thing that can fail
here, and a caller that cannot record is expected to carry on without
recording, so the error says which file it was.

A database that opens is kept even when the other does not: half a
history is worth more than none, and the report says which file each
half came from.

```go
func NewStorage() ([]*Storage, error)
```

### `NewStorageAt`

`NewStorageAt` opens the event database at a path, applies the
migrations and returns it. Memory opens a private database held on one
connection.

```go
func NewStorageAt(ctx context.Context, path string) (*Storage, error)
```

### `Normalize`

`Normalize` reduces a command to the form the statistics group by, and
returns it with the tool that leads it.

The form keeps the tool and its subcommands, keeps the flags with
their values removed, sorts the flags, and drops everything else:

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

Sorting is what makes `go test -v -cover` and `go test -cover -v` one
row. Dropping operands is what makes a flag's argument disappear with
no table of which flags take one: `--cpuprofile cpu.out` leaves the
flag and the file is an operand like any other. A flag that is only
digits goes with them, so `head -20` and `head -50` count together.

```go
func Normalize(command string) string
```

### `OpenAll`

`OpenAll` opens each path and applies the migrations, in order. It
returns what opened and the first error, so a caller may use the one
that worked and report the one that did not.

```go
func OpenAll(ctx context.Context, paths ...string) ([]*Storage, error)
```

### `Record`

`Record` stores a finished run, then drops whatever has fallen outside
the retention window.

Cleaning up on the run and not on a timer is what keeps atkins a
command and not a service: a run that records is a run that has the
database open, and a bounded delete against an index costs a
millisecond when there is nothing to remove.

```go
func (*Recorder) Record(ctx context.Context, log *eventlog.Log) error
```

### `Close`

`Close` releases the connection pool. A nil Storage closes nothing, so
a caller that failed to open one has no branch to write.

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

### `Path`

`Path` returns the file the storage opened, for a report that names it.

```go
func (*Storage) Path() string
```

### `Purge`

`Purge` removes the history that started before a time.

It deletes in bounded batches and leaves the rest for the next pass. A
single DELETE over a year of commands holds the write lock long enough
that every atkins run on the machine waits on it.

```go
func (*Storage) Purge(ctx context.Context, req PurgeRequest) (*PurgeResult, error)
```

### `RawStartedAt`

`RawStartedAt` is a probe used while settling the stored time format.

```go
func (*Storage) RawStartedAt(ctx context.Context) (string, error)
```

### `Record`

`Record` stores a finished run: a run row per node of the tree and a
command row per command the run executed.

The rows go in one transaction. A run is read as a whole - the job's
time against its steps' - and half of one recorded is worse than none,
because a report would quietly treat it as complete.

```go
func (*Storage) Record(ctx context.Context, req RecordRequest) (*RecordResult, error)
```

### `Recorder`

`Recorder` returns the storage as a recorder for the runner, recording
the directory it is called from. A nil Storage returns nil, and a nil
Recorder records nothing, so a caller that could not open the database
has no branch to write.

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

`Samples` returns the commands of the request one by one, oldest first.

The filter runs on the command table alone, in a subquery, because
the run rows it joins share its column names.

A command's job is the job node above its step, or the node it names
itself when a $() substitution ran outside any step.

```go
func (*Storage) Samples(ctx context.Context, req SampleRequest) ([]Sample, error)
```

### `Stats`

`Stats` reports the command keys by the time they have taken
altogether, the most expensive first.

The database groups by the normalized form and the report folds the
forms into their keys, so a row recorded before the key existed
counts under it without a migration.

```go
func (*Storage) Stats(ctx context.Context, req StatsRequest) ([]CommandStat, error)
```

### `Tools`

`Tools` reports the tools by the time they have taken altogether, the
most expensive first. They roll up the command keys, so a tool counts
the same commands the command report does.

```go
func (*Storage) Tools(ctx context.Context, req StatsRequest) ([]ToolStat, error)
```

### `Usage`

`Usage` reports runs and time per tool per bucket, oldest first, which
is the series a usage chart draws.

```go
func (*Storage) Usage(ctx context.Context, req UsageRequest) ([]UsageEntry, error)
```

### `List`

`List` returns the databases to open, the project's first. A run
records to each, and a report reads each.

```go
func (Paths) List() []string
```

### `Record`

`Record` stores a finished run in each database.

Every one is written even when an earlier fails, and the first error
is the one reported: a database that is locked does not cost the run
its row in the other.

```go
func (Recorders) Record(ctx context.Context, log *eventlog.Log) error
```
