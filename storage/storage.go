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

// Storage is one atkins event database: it applies the migrations on
// open, writes the rows of finished runs and answers the reports. Every
// method takes a context and one request value. A nil Storage records
// nothing and closes nothing, so a caller that failed to open one has no
// branch to write.
type Storage struct {
	db   *sqlx.DB
	path string
}

// NewStorage returns the databases Discover names for the working
// directory, as OpenAll opens them.
func NewStorage() ([]*Storage, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	paths, err := Discover(dir)
	if err != nil {
		return nil, err
	}

	return OpenAll(context.Background(), paths.List()...)
}

// OpenAll returns the databases that opened, in order, and the first error.
func OpenAll(ctx context.Context, paths ...string) ([]*Storage, error) {
	var (
		stores []*Storage
		failed error
	)
	for _, path := range paths {
		store, err := NewStorageAt(ctx, path)
		if err != nil {
			if failed == nil {
				failed = err
			}
			continue
		}
		stores = append(stores, store)
	}
	return stores, failed
}

// NewStorageAt returns the migrated database at path, Memory for a private one.
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

// Close releases the connection pool; a nil Storage returns nil.
func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// timeFormat makes the driver write a time in a layout strftime reads,
// `2006-01-02 15:04:05.999999999-07:00`. With time.Time's own String
// form every usage bucket comes back empty.
const timeFormat = "_time_format=sqlite"

// open returns the sqlite pool for path, with a busy timeout and the
// write-ahead log on a file. main and the tests link the driver.
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

// client returns a request scoped pdo client over the shared pool.
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

// newID returns a ULID for a row, which sorts by creation time.
func newID() string {
	return ulid.Make().String()
}

// stamp returns nil for a zero time, else the time in UTC to the
// millisecond, so a text column compares in one zone.
func stamp(at time.Time) *time.Time {
	if at.IsZero() {
		return nil
	}
	utc := at.UTC().Truncate(time.Millisecond)
	return &utc
}

// RawStartedAt returns the started_at of a command row as the database
// holds the text, or the error of an empty table.
func (s *Storage) RawStartedAt(ctx context.Context) (string, error) {
	client := s.client()
	defer client.Close()
	value, err := client.Get[string](ctx, "SELECT CAST(started_at AS TEXT) FROM command LIMIT 1")
	if err != nil {
		return "", err
	}
	return *value, nil
}

// Paths are the event databases a run records to, the project's and the
// workspace's. The package documentation says why a run writes both.
type Paths struct {
	// Project is the atkins.db of the nearest .atkins folder, empty
	// when the directory belongs to no project, and empty when the
	// nearest folder is the workspace's own.
	Project string

	// Workspace is $HOME/.atkins/atkins.db, which every run records to.
	Workspace string
}

// Discover returns the databases for dir and creates $HOME/.atkins. The
// project's is the atkins.db of the nearest .atkins folder above dir,
// and empty when that folder is the workspace's own or there is none.
func Discover(dir string) (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	workspace := filepath.Join(home, ".atkins")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return Paths{}, err
	}

	paths := Paths{Workspace: filepath.Join(workspace, "atkins.db")}

	for at := filepath.Clean(dir); ; {
		candidate := filepath.Join(at, ".atkins")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if project := filepath.Join(candidate, "atkins.db"); project != paths.Workspace {
				paths.Project = project
			}
			return paths, nil
		}

		parent := filepath.Dir(at)
		if parent == at {
			return paths, nil
		}
		at = parent
	}
}

// List returns the workspace alone without a project, else both, the project's first.
func (p Paths) List() []string {
	if p.Project == "" {
		return []string{p.Workspace}
	}
	return []string{p.Project, p.Workspace}
}
