package storage

import (
	"context"
	"strings"
	"time"

	"github.com/titpetric/atkins/model"
)

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
