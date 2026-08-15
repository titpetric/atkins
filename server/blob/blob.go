// Package blob stores the bytes of job artefacts.
//
// The database records what an artefact is — which job produced it,
// what the pipeline called it, how big it is and what it hashes to.
// This package holds the bytes themselves, addressed by an opaque key
// that the row carries.
//
// Splitting the two is the whole point. A row is small and worth
// keeping; bytes are large and are the thing that fills a disk. It also
// means the only thing an object store backend has to implement is this
// interface: Put, Open, Remove against a key namespace, with no SQL and
// no HTTP handler anywhere near it.
package blob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/titpetric/platform/pkg/ulid"
)

// Store holds artefact bytes under opaque keys.
//
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

// Result describes what a Put wrote.
type Result struct {
	// Size is the number of bytes stored.
	Size int64

	// Checksum is the SHA256 of those bytes, in lower case hex. It is
	// computed while writing rather than by reading the file back,
	// which would double the I/O for no extra confidence.
	Checksum string
}

// Errors returned by a Store.
var (
	// ErrTooLarge is returned when the reader had more to give than the
	// limit allowed.
	ErrTooLarge = errors.New("blob exceeds the maximum size")

	// ErrInvalidKey is returned for a key that cannot address a blob.
	ErrInvalidKey = errors.New("invalid blob key")
)

// Key builds the key an artefact's bytes live under.
//
// Grouping by job means retention can drop a job's bytes as a
// directory, and it keeps a single flat directory from growing to a
// million entries on a busy instance.
func Key(jobID, artefactID string) string {
	return path.Join(jobID, artefactID)
}

// Dir stores blobs as files under a root directory.
//
// It is the right first backend and probably the right one for most
// installs: the server already keeps its database on a disk, artefacts
// are written once and read rarely, and a directory is something an
// operator can back up, rsync or point a webserver at without asking
// anybody for a bucket.
type Dir struct {
	root string
}

// Verify contract.
var _ Store = (*Dir)(nil)

// NewDir returns a Store writing under root.
func NewDir(root string) *Dir {
	return &Dir{root: root}
}

// Root returns the directory blobs are written under.
func (d *Dir) Root() string {
	return d.root
}

// Prepare creates the root directory.
//
// The server calls this at start-up: a root it cannot write to should
// stop the process with a clear error, not surface as a failed upload
// on the first job that produces a file.
func (d *Dir) Prepare() error {
	return os.MkdirAll(d.root, 0o750)
}

// Put streams r into the store, hashing as it goes.
func (d *Dir) Put(_ context.Context, key string, r io.Reader, limit int64) (Result, error) {
	name, err := d.resolve(key)
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		return Result{}, err
	}

	// Write beside the target and rename: a reader can then only ever
	// see a complete blob, and an interrupted upload leaves a .part
	// file rather than a truncated artefact.
	temporary := name + "." + ulid.String() + ".part"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return Result{}, err
	}

	defer func() {
		file.Close()
		os.Remove(temporary)
	}()

	source := r
	if limit > 0 {
		// One byte past the limit: enough to tell "exactly at the
		// limit" from "more than we agreed to store".
		source = io.LimitReader(r, limit+1)
	}

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), source)
	if err != nil {
		return Result{}, err
	}
	if limit > 0 && size > limit {
		return Result{}, ErrTooLarge
	}

	if err := file.Close(); err != nil {
		return Result{}, err
	}
	if err := os.Rename(temporary, name); err != nil {
		return Result{}, err
	}

	return Result{Size: size, Checksum: hex.EncodeToString(hash.Sum(nil))}, nil
}

// Open returns the stored bytes.
func (d *Dir) Open(_ context.Context, key string) (io.ReadCloser, error) {
	name, err := d.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.Open(name)
}

// Remove deletes the stored bytes, and the job directory with them once
// it is empty.
func (d *Dir) Remove(_ context.Context, key string) error {
	name, err := d.resolve(key)
	if err != nil {
		return err
	}

	if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Best effort: this fails while the job has other artefacts, which
	// is the normal case and not worth reporting.
	_ = os.Remove(filepath.Dir(name))

	return nil
}

// resolve turns a key into a path under the root, refusing anything
// that would address a file outside it.
func (d *Dir) resolve(key string) (string, error) {
	key = strings.TrimSpace(strings.ReplaceAll(key, `\`, "/"))
	if key == "" || strings.HasPrefix(key, "/") {
		return "", ErrInvalidKey
	}

	clean := path.Clean(key)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", ErrInvalidKey
	}

	return filepath.Join(d.root, filepath.FromSlash(clean)), nil
}
