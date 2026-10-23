package storage

import (
	"context"
	"sort"
)

// CommandUsage is one command key in one bucket of time: the point a
// chart stacked by command draws.
type CommandUsage struct {
	// Bucket is the start of the window, as the database spells a time.
	Bucket string

	// Command is the key, as Key names it, and Tool its first word.
	Command string
	Tool    string

	// Runs and TotalMs are how often it ran in the window and for how
	// long altogether.
	Runs    int64
	TotalMs int64
}

// commandUsageRow is one normalized form in one bucket, which the
// report folds into its key.
type commandUsageRow struct {
	Bucket     string `db:"bucket"`
	Normalized string `db:"normalized"`
	Runs       int64  `db:"runs"`
	TotalMs    int64  `db:"total_ms"`
}

// CommandUsage reports runs and time per command key per bucket, oldest
// first and the most expensive first within a bucket.
func (s *Storage) CommandUsage(ctx context.Context, req UsageRequest) ([]CommandUsage, error) {
	expression, err := bucketExpression(req.Bucket)
	if err != nil {
		return nil, err
	}

	where, args := commandWhere(req.Since, req.Until, req.Tool, req.Dir)

	query := `SELECT
			` + expression + ` AS bucket,
			normalized,
			COUNT(*) AS runs,
			SUM(duration_ms) AS total_ms
		FROM command` + where + `
		GROUP BY bucket, normalized`

	client := s.client()
	defer client.Close()

	rows, err := client.Select[commandUsageRow](ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var (
		order []string
		byKey = map[string]*CommandUsage{}
	)
	for _, row := range rows {
		key, tool := Key(row.Normalized)
		if key == "" {
			continue
		}
		id := row.Bucket + "\x00" + key
		entry, ok := byKey[id]
		if !ok {
			entry = &CommandUsage{Bucket: row.Bucket, Command: key, Tool: tool}
			byKey[id] = entry
			order = append(order, id)
		}
		entry.Runs += row.Runs
		entry.TotalMs += row.TotalMs
	}

	usage := make([]CommandUsage, 0, len(order))
	for _, id := range order {
		usage = append(usage, *byKey[id])
	}
	sort.SliceStable(usage, func(i, j int) bool {
		if usage[i].Bucket != usage[j].Bucket {
			return usage[i].Bucket < usage[j].Bucket
		}
		if usage[i].TotalMs != usage[j].TotalMs {
			return usage[i].TotalMs > usage[j].TotalMs
		}
		return usage[i].Command < usage[j].Command
	})

	return usage, nil
}
