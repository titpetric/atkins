package storage

import (
	"context"
	"time"
)

// Count returns how many runs and commands started before a time, every
// row for a zero time: what a prune would remove, or what remains.
func (s *Storage) Count(ctx context.Context, before time.Time) (*PurgeResult, error) {
	client := s.client()
	defer client.Close()

	result := &PurgeResult{}
	for _, table := range []struct {
		name  string
		count *int64
	}{
		{"run", &result.Runs},
		{"command", &result.Commands},
	} {
		query, args := "SELECT COUNT(*) FROM "+table.name, []any{}
		if stamp := stamp(before); stamp != nil {
			query, args = query+" WHERE started_at < ?", []any{*stamp}
		}
		n, err := client.Get[int64](ctx, query, args...)
		if err != nil {
			return nil, err
		}
		*table.count = *n
	}
	return result, nil
}
