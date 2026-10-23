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
