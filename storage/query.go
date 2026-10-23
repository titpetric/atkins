package storage

import (
	"context"
	"strings"
	"time"

	"github.com/titpetric/atkins/model"
)

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

// Runs lists run rows, newest first.
func (s *Storage) Runs(ctx context.Context, filter RunFilter) ([]model.Run, error) {
	var (
		clauses = []string{"1 = 1"}
		args    []any
	)

	if filter.Invocation != "" {
		clauses = append(clauses, "invocation_id = ?")
		args = append(args, filter.Invocation)
	}
	if filter.Kind != "" {
		clauses = append(clauses, "kind = ?")
		args = append(args, filter.Kind)
	}
	if filter.Name != "" {
		clauses = append(clauses, "name = ?")
		args = append(args, filter.Name)
	}
	if at := stamp(filter.Since); at != nil {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, *at)
	}
	if at := stamp(filter.Until); at != nil {
		clauses = append(clauses, "started_at <= ?")
		args = append(args, *at)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	query := "SELECT * FROM run WHERE " + strings.Join(clauses, " AND ") +
		" ORDER BY started_at DESC, id DESC LIMIT ?"

	client := s.client()
	defer client.Close()

	return client.Select[model.Run](ctx, query, append(args, limit)...)
}

// Run returns one run row by ID, or ErrNotFound.
func (s *Storage) Run(ctx context.Context, id string) (*model.Run, error) {
	client := s.client()
	defer client.Close()

	run, err := client.Get[model.Run](ctx, "SELECT * FROM run WHERE id = ?", id)
	return run, notFound(err)
}

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

// Commands lists command rows, newest first.
func (s *Storage) Commands(ctx context.Context, filter CommandFilter) ([]model.Command, error) {
	var (
		clauses = []string{"1 = 1"}
		args    []any
	)

	if filter.Invocation != "" {
		clauses = append(clauses, "invocation_id = ?")
		args = append(args, filter.Invocation)
	}
	if filter.Parent != "" {
		clauses = append(clauses, "parent_id = ?")
		args = append(args, filter.Parent)
	}
	if filter.Tool != "" {
		clauses = append(clauses, "tool = ?")
		args = append(args, filter.Tool)
	}
	if filter.Normalized != "" {
		clauses = append(clauses, "normalized = ?")
		args = append(args, filter.Normalized)
	}
	if at := stamp(filter.Since); at != nil {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, *at)
	}
	if at := stamp(filter.Until); at != nil {
		clauses = append(clauses, "started_at <= ?")
		args = append(args, *at)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	query := "SELECT * FROM command WHERE " + strings.Join(clauses, " AND ") +
		" ORDER BY started_at DESC, id DESC LIMIT ?"

	client := s.client()
	defer client.Close()

	return client.Select[model.Command](ctx, query, append(args, limit)...)
}

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

// PurgeResult says how many rows of each kind went.
type PurgeResult struct {
	Runs     int64
	Commands int64
}

// Purge removes the history that started before a time.
//
// It deletes in bounded batches and leaves the rest for the next pass. A
// single DELETE over a year of commands holds the write lock long enough
// that every atkins run on the machine waits on it.
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

// RawStartedAt is a probe used while settling the stored time format.
func (s *Storage) RawStartedAt(ctx context.Context) (string, error) {
	client := s.client()
	defer client.Close()
	value, err := client.Get[string](ctx, "SELECT CAST(started_at AS TEXT) FROM command LIMIT 1")
	if err != nil {
		return "", err
	}
	return *value, nil
}
