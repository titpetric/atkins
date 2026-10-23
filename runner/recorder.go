package runner

import (
	"context"

	"github.com/titpetric/atkins/eventlog"
)

// Recorder stores what a finished run did. The event log document is
// the whole of it: the metadata, the tree, every event and the summary.
//
// The interface keeps the runner free of a database: a sqlite file, a
// socket or a test double all satisfy it, and a recorder that fails
// never fails the run that produced it.
type Recorder interface {
	Record(ctx context.Context, log *eventlog.Log) error
}
