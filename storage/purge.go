package storage

import (
	"context"
	"time"
)

// PurgeRequest is how much history to drop.
type PurgeRequest struct {
	// Before removes everything that started before this time. A zero
	// time removes nothing.
	Before time.Time

	// Batch bounds one DELETE and Batches how many it runs. A first
	// pass over a year of history holds the write lock for a moment.
	// Zero means 500 rows and 20 batches, and the next pass takes the
	// rows beyond them.
	Batch   int
	Batches int
}

// PurgeResult counts the rows of each kind a Purge removed.
type PurgeResult struct {
	Runs     int64
	Commands int64
}

// Purge returns an empty result for a zero time, an error from a DELETE,
// or the rows it removed, in bounded batches, of the history started
// before req.Before.
func (s *Storage) Purge(ctx context.Context, req PurgeRequest) (*PurgeResult, error) {
	result := &PurgeResult{}
	before := stamp(req.Before)
	if before == nil {
		return result, nil
	}

	batch := req.Batch
	if batch <= 0 {
		batch = 500
	}
	batches := req.Batches
	if batches <= 0 {
		batches = 20
	}

	client := s.client()
	defer client.Close()

	for _, table := range []struct {
		name  string
		count *int64
	}{
		{"command", &result.Commands},
		{"run", &result.Runs},
	} {
		query := "DELETE FROM " + table.name +
			" WHERE id IN (SELECT id FROM " + table.name + " WHERE started_at < ? LIMIT ?)"

		for range batches {
			if err := client.Exec(ctx, query, *before, batch); err != nil {
				return nil, err
			}
			affected := client.RowsAffected()
			*table.count += affected
			if affected < int64(batch) {
				break
			}
		}
	}

	return result, nil
}
