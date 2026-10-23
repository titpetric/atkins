package storage

import (
	"context"
	"fmt"
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

// CommandStat is one command key and what it has cost.
type CommandStat struct {
	// Command is the key the rows group by, as Key names it: `go test`
	// counts every test run whatever its flags and packages.
	Command string

	// Tool is the first word of it.
	Tool string

	// Runs is how many times it ran.
	Runs int64

	// TotalMs is the time it has taken altogether, which is what the
	// report sorts by: a command that takes a second and runs four
	// hundred times costs more than a build that takes a minute.
	TotalMs int64

	// AvgMs and MaxMs are the average and the worst single run.
	AvgMs int64
	MaxMs int64

	// Failures is how many of the runs exited non-zero.
	Failures int64

	// LastAt is when it last ran.
	LastAt *time.Time
}

// statRow is one normalized form as the query returns it, which the
// report folds into its key.
//
// MAX(started_at) is an expression, and an expression has no declared
// column type for the driver to convert by, so the newest time arrives
// as the text the database holds and is parsed here.
type statRow struct {
	Normalized string `db:"normalized"`
	Runs       int64  `db:"runs"`
	TotalMs    int64  `db:"total_ms"`
	MaxMs      int64  `db:"max_ms"`
	Failures   int64  `db:"failures"`
	LastAt     string `db:"last_at"`
}

// Stats reports the command keys by the time they have taken
// altogether, the most expensive first.
//
// The database groups by the normalized form and the report folds the
// forms into their keys, so a row recorded before the key existed
// counts under it without a migration.
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
		// Rounded, not truncated: a command averaging under a
		// millisecond has still taken time, and a column of zeroes
		// reads as a measurement that failed.
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
	// Tool is the first word of the command key.
	Tool string

	// Commands is how many distinct command keys of it ran.
	Commands int64

	// Runs is how many times any of them ran.
	Runs int64

	// TotalMs is the time they have taken altogether.
	TotalMs int64

	// Failures is how many of the runs exited non-zero.
	Failures int64
}

// Tools reports the tools by the time they have taken altogether, the
// most expensive first. They roll up the command keys, so a tool counts
// the same commands the command report does.
func (s *Storage) Tools(ctx context.Context, req StatsRequest) ([]ToolStat, error) {
	stats, err := s.keyed(ctx, req)
	if err != nil {
		return nil, err
	}

	var (
		order  []string
		byTool = map[string]*ToolStat{}
	)
	for _, stat := range stats {
		tool, ok := byTool[stat.Tool]
		if !ok {
			tool = &ToolStat{Tool: stat.Tool}
			byTool[stat.Tool] = tool
			order = append(order, stat.Tool)
		}
		tool.Commands++
		tool.Runs += stat.Runs
		tool.TotalMs += stat.TotalMs
		tool.Failures += stat.Failures
	}

	tools := make([]ToolStat, 0, len(order))
	for _, name := range order {
		tools = append(tools, *byTool[name])
	}
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].TotalMs != tools[j].TotalMs {
			return tools[i].TotalMs > tools[j].TotalMs
		}
		if tools[i].Runs != tools[j].Runs {
			return tools[i].Runs > tools[j].Runs
		}
		return tools[i].Tool < tools[j].Tool
	})

	return tools[:min(len(tools), limitOf(req.Limit))], nil
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

	// A builtin is the shell setting itself up. The
	// clause also leaves out the builtin rows of older databases.
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
