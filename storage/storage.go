// Package storage owns the atkins event database.
//
// Every run leaves two kinds of row behind: a `run` row per node of the
// tree atkins walked, carrying the time that node took with everything
// under it, and a `command` row per command a step actually executed.
// The first answers "how long does the release job take", the second
// "how much of my day is go test".
//
// The file is atkins.db in the nearest .atkins folder, and
// $HOME/.atkins/atkins.db when a directory belongs to no project. One
// type, Storage, owns all of it: every method takes a context and one
// request value and returns one result and an error, so a reader - the
// --stats report today, task-ui later - has one surface to learn.
//
// Queries go through github.com/titpetric/pdo, whose generic methods
// scan straight into the mig generated types in ../model. A *pdo.PDO is
// request scoped and not safe for concurrent use, so every method
// allocates one over the shared pool.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-bridget/mig/migrate"
	"github.com/jmoiron/sqlx"
	ulid "github.com/oklog/ulid/v2"
	"github.com/titpetric/pdo"

	"github.com/titpetric/atkins/schema"
)

// Project is the migration project name recorded in the migrations
// table.
const Project = "atkins"

// Memory is the DSN of a private in-memory database, for tests and for
// a run that must record nowhere.
const Memory = ":memory:"

// ErrNotFound reports a row that is not there.
var ErrNotFound = errors.New("not found")

// Storage is the atkins event database.
type Storage struct {
	db   *sqlx.DB
	path string
}

// NewStorage opens the event database for the working directory,
// applies the migrations and returns it.
//
// The path comes from Discover: the nearest .atkins folder, or
// $HOME/.atkins. Opening is the only thing that can fail here, and a
// caller that cannot record is expected to carry on without recording,
// so the error says which file it was.
func NewStorage() (*Storage, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path, err := Discover(dir)
	if err != nil {
		return nil, err
	}

	return NewStorageAt(context.Background(), path)
}

// NewStorageAt opens the event database at a path, applies the
// migrations and returns it. Memory opens a private database held on one
// connection.
func NewStorageAt(ctx context.Context, path string) (*Storage, error) {
	db, err := open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	manager, err := migrate.NewManager(db, schema.Migrations(), Project)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if _, err := manager.Apply(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &Storage{db: db, path: path}, nil
}

// Path returns the file the storage opened, for a report that names it.
func (s *Storage) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Close releases the connection pool. A nil Storage closes nothing, so
// a caller that failed to open one has no branch to write.
func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// timeFormat makes the driver write a time in a layout SQLite
// understands: `2006-01-02 15:04:05.999999999-07:00`.
//
// Without it the driver writes time.Time's own String form,
// `2026-03-01 12:00:00 +0000 UTC`, which strftime cannot read, so every
// usage bucket would come back empty. The usage report is the reason the
// database exists, so the parameter is not optional.
const timeFormat = "_time_format=sqlite"

// open opens the sqlite database. The driver is the program's to link:
// main imports modernc.org/sqlite for its side effect, as do the tests.
//
// A file also gets a busy timeout and the write-ahead log, because
// several atkins runs in one checkout write to it at once and the second
// must wait rather than fail.
func open(path string) (*sqlx.DB, error) {
	memory := path == "" || path == Memory
	dsn := path
	switch {
	case memory:
		dsn = Memory + "?" + timeFormat
	case strings.Contains(dsn, "?"):
		dsn += "&" + timeFormat
	default:
		dsn += "?" + timeFormat + "&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if memory {
		db.SetMaxOpenConns(1)
	}
	return db, nil
}

// client returns a request scoped pdo client over the shared pool. It is
// the one place pdo is named, so the rest of the package reads as SQL and
// model types.
func (s *Storage) client() *pdo.PDO {
	return pdo.New(s.db)
}

// notFound maps the driver's missing row to ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// newID returns an identifier for a row: a ULID, 26 characters, sorting
// by the moment it was made.
func newID() string {
	return ulid.Make().String()
}

// stamp returns a time as the database keeps it, in UTC to the
// millisecond. A column compared or ordered as text holds one zone,
// whatever the zone of the clock that made the value. Nil stays nil.
func stamp(at time.Time) *time.Time {
	if at.IsZero() {
		return nil
	}
	utc := at.UTC().Truncate(time.Millisecond)
	return &utc
}

// Discover returns the path of the event database for a directory.
//
// It walks up from dir looking for a .atkins folder and answers with the
// atkins.db inside the first one it finds, so every run in a checkout
// records to the same file. A directory that belongs to no project
// records to $HOME/.atkins/atkins.db, which Discover creates.
func Discover(dir string) (string, error) {
	for at := filepath.Clean(dir); ; {
		candidate := filepath.Join(at, ".atkins")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Join(candidate, "atkins.db"), nil
		}

		parent := filepath.Dir(at)
		if parent == at {
			break
		}
		at = parent
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	fallback := filepath.Join(home, ".atkins")
	if err := os.MkdirAll(fallback, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(fallback, "atkins.db"), nil
}
