package storage

import (
	"context"
	"os"
	"time"

	"github.com/titpetric/atkins/eventlog"
)

// Retention is how long a recorded run is kept, measured from when it
// started.
//
// Thirty-one days, fixed. It is long enough to hold a month of work and
// compare this week against the last, and short enough that a checkout
// nobody prunes does not grow without bound. A window nobody has to
// choose is a window that is right on every machine, including the ones
// where the configuration was never opened.
const Retention = 31 * 24 * time.Hour

// Recorder stores finished runs. It satisfies runner.Recorder, which is
// the only thing the runner knows about this package: the runner hands
// over an event log document and never names a table or a column.
type Recorder struct {
	storage *Storage
	dir     string
}

// Recorder returns the storage as a recorder for the runner, recording
// the directory it is called from. A nil Storage returns nil, and a nil
// Recorder records nothing, so a caller that could not open the database
// has no branch to write.
func (s *Storage) Recorder() *Recorder {
	if s == nil {
		return nil
	}

	dir, _ := os.Getwd()
	return &Recorder{storage: s, dir: dir}
}

// Record stores a finished run, then drops whatever has fallen outside
// the retention window.
//
// Cleaning up on the run and not on a timer is what keeps atkins a
// command and not a service: a run that records is a run that has the
// database open, and a bounded delete against an index costs a
// millisecond when there is nothing to remove.
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
