package storage

import (
	"context"
	"fmt"
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

// CommandStat is one normalized command and what it has cost.
type CommandStat struct {
	// Normalized is the command with its operands dropped and its flags
	// sorted, the form the rows group by.
	Normalized string `db:"normalized"`

	// Tool is the first word of it.
	Tool string `db:"tool"`

	// Runs is how many times it ran.
	Runs int64 `db:"runs"`

	// TotalMs is the time it has taken altogether, which is what the
	// report sorts by: a command that takes a second and runs four
	// hundred times costs more than a build that takes a minute.
	TotalMs int64 `db:"total_ms"`

	// AvgMs and MaxMs are the average and the worst single run.
	AvgMs int64 `db:"avg_ms"`
	MaxMs int64 `db:"max_ms"`

	// Failures is how many of the runs exited non-zero.
	Failures int64 `db:"failures"`

	// LastAt is when it last ran.
	LastAt *time.Time
}

// statRow is CommandStat as the query returns it.
//
// MAX(started_at) is an expression, and an expression has no declared
// column type for the driver to convert by, so the newest time arrives
// as the text the database holds and is parsed here.
type statRow struct {
	CommandStat
	LastAt string `db:"last_at"`
}

// Stats reports the normalized commands by the time they have taken
// altogether, the most expensive first.
func (s *Storage) Stats(ctx context.Context, req StatsRequest) ([]CommandStat, error) {
	where, args := commandWhere(req.Since, req.Until, req.Tool, req.Dir)
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	query := `SELECT
			normalized,
			tool,
			COUNT(*) AS runs,
			SUM(duration_ms) AS total_ms,
			-- Rounded, not truncated: a command averaging under a
			-- millisecond has still taken time, and a column of zeroes
			-- reads as a measurement that failed.
			CAST(AVG(duration_ms) + 0.5 AS INTEGER) AS avg_ms,
			MAX(duration_ms) AS max_ms,
			SUM(CASE WHEN exit_code != 0 THEN 1 ELSE 0 END) AS failures,
			MAX(started_at) AS last_at
		FROM command` + where + `
		GROUP BY normalized
		ORDER BY total_ms DESC, runs DESC
		LIMIT ?`

	client := s.client()
	defer client.Close()

	rows, err := client.Select[statRow](ctx, query, append(args, limit)...)
	if err != nil {
		return nil, err
	}

	stats := make([]CommandStat, 0, len(rows))
	for _, row := range rows {
		stat := row.CommandStat
		stat.LastAt = parseStamp(row.LastAt)
		stats = append(stats, stat)
	}

	return stats, nil
}

// parseStamp reads a time back out of the text the database holds. An
// unparseable value is no time rather than an error: a report is worth
// printing without its last-seen column.
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

// ToolStat is one tool and what it has cost, across every form of it.
//
// It answers the question a per-command report cannot: `go` is one tool
// whether it builds, tests or vets, and the share of a week it takes is
// the share of a week spent on Go.
type ToolStat struct {
	// Tool is the first word of the normalized command.
	Tool string `db:"tool"`

	// Commands is how many distinct normalized forms of it ran.
	Commands int64 `db:"commands"`

	// Runs is how many times any of them ran.
	Runs int64 `db:"runs"`

	// TotalMs is the time they have taken altogether.
	TotalMs int64 `db:"total_ms"`

	// Failures is how many of the runs exited non-zero.
	Failures int64 `db:"failures"`
}

// Tools reports the tools by the time they have taken altogether, the
// most expensive first.
func (s *Storage) Tools(ctx context.Context, req StatsRequest) ([]ToolStat, error) {
	where, args := commandWhere(req.Since, req.Until, req.Tool, req.Dir)
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	query := `SELECT
			tool,
			COUNT(DISTINCT normalized) AS commands,
			COUNT(*) AS runs,
			SUM(duration_ms) AS total_ms,
			SUM(CASE WHEN exit_code != 0 THEN 1 ELSE 0 END) AS failures
		FROM command` + where + `
		GROUP BY tool
		ORDER BY total_ms DESC, runs DESC
		LIMIT ?`

	client := s.client()
	defer client.Close()

	return client.Select[ToolStat](ctx, query, append(args, limit)...)
}

// Bucket is how wide a usage bucket is.
type Bucket string

// The buckets a usage report can ask for.
const (
	BucketHour Bucket = "hour"
	BucketDay  Bucket = "day"
	BucketWeek Bucket = "week"
)

// UsageRequest asks for usage over time, a series per tool.
type UsageRequest struct {
	// Since and Until bound when a command started.
	Since time.Time
	Until time.Time

	// Bucket is how wide one point is. Empty means a day.
	Bucket Bucket

	// Tool keeps the commands of one tool. Empty reports every tool.
	Tool string

	// Dir keeps the commands that ran in a directory or below it.
	Dir string
}

// UsageEntry is one tool in one bucket of time: the point a chart draws.
type UsageEntry struct {
	// Bucket is the start of the window, as the database spells a time.
	Bucket string `db:"bucket"`

	// Tool is the tool the point counts.
	Tool string `db:"tool"`

	// Runs and TotalMs are how often it ran in the window and for how
	// long altogether.
	Runs    int64 `db:"runs"`
	TotalMs int64 `db:"total_ms"`
}

// Usage reports runs and time per tool per bucket, oldest first, which
// is the series a usage chart draws.
func (s *Storage) Usage(ctx context.Context, req UsageRequest) ([]UsageEntry, error) {
	expression, err := bucketExpression(req.Bucket)
	if err != nil {
		return nil, err
	}

	where, args := commandWhere(req.Since, req.Until, req.Tool, req.Dir)

	query := `SELECT
			` + expression + ` AS bucket,
			tool,
			COUNT(*) AS runs,
			SUM(duration_ms) AS total_ms
		FROM command` + where + `
		GROUP BY bucket, tool
		ORDER BY bucket, total_ms DESC`

	client := s.client()
	defer client.Close()

	return client.Select[UsageEntry](ctx, query, args...)
}

// bucketExpression returns the SQL that names the window a command
// falls in.
//
// Every bucket is labelled with the date or hour it starts, rather than
// a week number, so a chart can plot the label on a time axis without
// decoding it.
func bucketExpression(bucket Bucket) (string, error) {
	switch bucket {
	case BucketHour:
		return "strftime('%Y-%m-%dT%H:00', started_at)", nil
	case "", BucketDay:
		return "date(started_at)", nil
	case BucketWeek:
		// The Monday of the command's week: back up six days, then
		// forward to the next Monday, which is a no-op on a Monday.
		return "date(started_at, '-6 days', 'weekday 1')", nil
	default:
		return "", fmt.Errorf("unknown bucket %q: hour, day or week", bucket)
	}
}

// commandWhere builds the WHERE clause the two reports share.
func commandWhere(since, until time.Time, tool, dir string) (string, []any) {
	var (
		clauses []string
		args    []any
	)

	// A command with no normalized form ran nothing a report can name:
	// an empty command line, or one that was only a shell operator.
	clauses = append(clauses, "normalized != ''")

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
