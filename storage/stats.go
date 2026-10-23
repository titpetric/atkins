package storage

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StatsRequest narrows the statistics. A zero value reports every
// command the database holds, the slowest first.
type StatsRequest struct {
	// Since and Until bound when a command started. A zero time is no
	// bound at that end.
	Since time.Time
	Until time.Time

	// Tool keeps the commands of one tool, as Normalize names it.
	Tool string

	// Dir keeps the commands that ran in a directory or below it.
	Dir string

	// Limit caps the rows. Zero means twenty, which is a screen.
	Limit int
}

// statRow is one normalized form as the query returns it, for the report
// to fold into its key. MAX(started_at) has no declared column type for
// the driver to convert by, so the newest time arrives as text and
// parseStamp reads it.
type statRow struct {
	Normalized string `db:"normalized"`
	Runs       int64  `db:"runs"`
	TotalMs    int64  `db:"total_ms"`
	MaxMs      int64  `db:"max_ms"`
	Failures   int64  `db:"failures"`
	LastAt     string `db:"last_at"`
}

// Stats returns the query error, or the command keys by the time they
// have taken altogether, the most expensive first.
func (s *Storage) Stats(ctx context.Context, req StatsRequest) ([]CommandStat, error) {
	stats, err := s.keyed(ctx, req)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].TotalMs != stats[j].TotalMs {
			return stats[i].TotalMs > stats[j].TotalMs
		}
		if stats[i].Runs != stats[j].Runs {
			return stats[i].Runs > stats[j].Runs
		}
		return stats[i].Command < stats[j].Command
	})

	return stats[:min(len(stats), limitOf(req.Limit))], nil
}

// keyed returns every command key of the request, unsorted.
func (s *Storage) keyed(ctx context.Context, req StatsRequest) ([]CommandStat, error) {
	where, args := commandWhere(req.Since, req.Until, req.Tool, req.Dir)

	query := `SELECT
			normalized,
			COUNT(*) AS runs,
			SUM(duration_ms) AS total_ms,
			MAX(duration_ms) AS max_ms,
			SUM(CASE WHEN exit_code != 0 THEN 1 ELSE 0 END) AS failures,
			MAX(started_at) AS last_at
		FROM command` + where + `
		GROUP BY normalized`

	client := s.client()
	defer client.Close()

	rows, err := client.Select[statRow](ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var (
		order []string
		byKey = map[string]*CommandStat{}
		last  = map[string]string{}
	)
	for _, row := range rows {
		key, tool := Key(row.Normalized)
		if key == "" {
			continue
		}
		stat, ok := byKey[key]
		if !ok {
			stat = &CommandStat{Command: key, Tool: tool}
			byKey[key] = stat
			order = append(order, key)
		}
		stat.Runs += row.Runs
		stat.TotalMs += row.TotalMs
		stat.MaxMs = max(stat.MaxMs, row.MaxMs)
		stat.Failures += row.Failures
		if row.LastAt > last[key] {
			last[key] = row.LastAt
		}
	}

	stats := make([]CommandStat, 0, len(order))
	for _, key := range order {
		stat := byKey[key]
		// Rounded up from half: a command averaging under a
		// millisecond has still taken time, and a column of zeroes
		// reads as a failed measurement.
		if stat.Runs > 0 {
			stat.AvgMs = (2*stat.TotalMs + stat.Runs) / (2 * stat.Runs)
		}
		stat.LastAt = parseStamp(last[key])
		stats = append(stats, *stat)
	}

	return stats, nil
}

// limitOf is a request's row cap. Zero means twenty, which is a screen.
func limitOf(limit int) int {
	if limit <= 0 {
		return 20
	}
	return limit
}

// parseStamp returns nil for an empty value, the time of the first
// layout that parses it, and nil for text no layout reads: a report
// prints without its last-seen column.
func parseStamp(value string) *time.Time {
	if value == "" {
		return nil
	}

	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if at, err := time.Parse(layout, value); err == nil {
			return &at
		}
	}

	return nil
}

// commandWhere builds the WHERE clause the two reports share.
func commandWhere(since, until time.Time, tool, dir string) (string, []any) {
	var (
		clauses []string
		args    []any
	)

	// A command with no normalized form ran nothing a report can name:
	// an empty command line, or a lone shell operator.
	clauses = append(clauses, "normalized != ''")

	// A builtin is the shell setting itself up. The clause also leaves
	// out the builtin rows of older databases.
	marks := make([]string, 0, len(builtins))
	for _, builtin := range builtins {
		marks = append(marks, "?")
		args = append(args, builtin)
	}
	clauses = append(clauses, "tool NOT IN ("+strings.Join(marks, ", ")+")")

	if at := stamp(since); at != nil {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, *at)
	}
	if at := stamp(until); at != nil {
		clauses = append(clauses, "started_at <= ?")
		args = append(args, *at)
	}
	if tool != "" {
		clauses = append(clauses, "tool = ?")
		args = append(args, tool)
	}
	if dir != "" {
		clauses = append(clauses, "(dir = ? OR dir LIKE ?)")
		args = append(args, dir, strings.TrimRight(dir, "/")+"/%")
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}
