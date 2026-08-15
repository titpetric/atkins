# Package ./server/blob

```go
import (
	"github.com/titpetric/atkins/server/blob"
}
```

Package blob stores the bytes of job artefacts.

The database records what an artefact is — which job produced it,
what the pipeline called it, how big it is and what it hashes to.
This package holds the bytes themselves, addressed by an opaque key
that the row carries.

Splitting the two is the whole point. A row is small and worth
keeping; bytes are large and are the thing that fills a disk. It also
means the only thing an object store backend has to implement is this
interface: Put, Open, Remove against a key namespace, with no SQL and
no HTTP handler anywhere near it.

## Types

```go
// Dir stores blobs as files under a root directory.
// It is the right first backend and probably the right one for most
// installs: the server already keeps its database on a disk, artefacts
// are written once and read rarely, and a directory is something an
// operator can back up, rsync or point a webserver at without asking
// anybody for a bucket.
type Dir struct {
	root string
}
```

```go
// Result describes what a Put wrote.
type Result struct {
	// Size is the number of bytes stored.
	Size int64

	// Checksum is the SHA256 of those bytes, in lower case hex. It is
	// computed while writing rather than by reading the file back,
	// which would double the I/O for no extra confidence.
	Checksum string
}
```

```go
// Store holds artefact bytes under opaque keys.
// A key is a slash separated path in the store's own namespace, not a
// filesystem path and not the name the pipeline used. Callers build it;
// the store only refuses one it cannot address safely.
type Store interface {
	// Put writes r under key, stopping at limit bytes, and reports what
	// was written. A limit of zero or less is unbounded.
	//
	// Nothing is readable under key until the write completes, so a
	// failed or oversized upload never leaves half an artefact behind.
	Put(ctx context.Context, key string, r io.Reader, limit int64) (Result, error)

	// Open returns the bytes stored under key.
	Open(ctx context.Context, key string) (io.ReadCloser, error)

	// Remove deletes the bytes stored under key. Removing a key that
	// isn't there is not an error: retention has to be idempotent.
	Remove(ctx context.Context, key string) error
}
```

## Vars

```go
// Errors returned by a Store.
var (
	// ErrTooLarge is returned when the reader had more to give than the
	// limit allowed.
	ErrTooLarge = errors.New("blob exceeds the maximum size")

	// ErrInvalidKey is returned for a key that cannot address a blob.
	ErrInvalidKey = errors.New("invalid blob key")
)
```

## Function symbols

- `func Key (jobID,artefactID string) string`
- `func NewDir (root string) *Dir`
- `func (*Dir) Open (_ context.Context, key string) (io.ReadCloser, error)`
- `func (*Dir) Prepare () error`
- `func (*Dir) Put (_ context.Context, key string, r io.Reader, limit int64) (Result, error)`
- `func (*Dir) Remove (_ context.Context, key string) error`
- `func (*Dir) Root () string`

### Key

Key builds the key an artefact's bytes live under.

Grouping by job means retention can drop a job's bytes as a
directory, and it keeps a single flat directory from growing to a
million entries on a busy instance.

```go
func Key(jobID, artefactID string) string
```

### NewDir

NewDir returns a Store writing under root.

```go
func NewDir(root string) *Dir
```

### Open

Open returns the stored bytes.

```go
func (*Dir) Open(_ context.Context, key string) (io.ReadCloser, error)
```

### Prepare

Prepare creates the root directory.

The server calls this at start-up: a root it cannot write to should
stop the process with a clear error, not surface as a failed upload
on the first job that produces a file.

```go
func (*Dir) Prepare() error
```

### Put

Put streams r into the store, hashing as it goes.

```go
func (*Dir) Put(_ context.Context, key string, r io.Reader, limit int64) (Result, error)
```

### Remove

Remove deletes the stored bytes, and the job directory with them once
it is empty.

```go
func (*Dir) Remove(_ context.Context, key string) error
```

### Root

Root returns the directory blobs are written under.

```go
func (*Dir) Root() string
```
