package storage

import (
	"context"
	"os"
	"time"

	"github.com/titpetric/atkins/eventlog"
)

// Retention is how long the database holds a run, from its start:
// thirty-one days, fixed. The package documentation says why.
const Retention = 31 * 24 * time.Hour

// Recorder stores finished runs into one Storage and purges what falls
// outside Retention. It implements runner.Recorder, the one view the
// runner has of this package: it passes an event log document and never
// names a table or a column. A nil Recorder records nothing.
type Recorder struct {
	storage *Storage
	dir     string
}

// Recorder returns nil for a nil Storage, else a recorder of the working
// directory.
func (s *Storage) Recorder() *Recorder {
	if s == nil {
		return nil
	}

	dir, _ := os.Getwd()
	return &Recorder{storage: s, dir: dir}
}

// Recorders writes a run to each database, the project's and the
// workspace's, the same rows in both. A failed write leaves the others
// to go ahead.
type Recorders []*Recorder

// NewRecorders returns a recorder per storage.
func NewRecorders(stores []*Storage) Recorders {
	recorders := make(Recorders, 0, len(stores))
	for _, store := range stores {
		recorders = append(recorders, store.Recorder())
	}
	return recorders
}

// Record returns the first error after writing to every database.
func (r Recorders) Record(ctx context.Context, log *eventlog.Log) error {
	var failed error
	for _, recorder := range r {
		if err := recorder.Record(ctx, log); err != nil && failed == nil {
			failed = err
		}
	}
	return failed
}

// Record returns nil on a nil Recorder, the write error, or the error
// of purging the history older than Retention.
func (r *Recorder) Record(ctx context.Context, log *eventlog.Log) error {
	if r == nil {
		return nil
	}

	if _, err := r.storage.Record(ctx, RecordRequest{Log: log, Dir: r.dir}); err != nil {
		return err
	}

	_, err := r.storage.Purge(ctx, PurgeRequest{Before: time.Now().Add(-Retention)})
	return err
}
