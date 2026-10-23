package storage

import (
	"context"
	"fmt"
	"time"
)

// Bucket is how wide a usage bucket is.
type Bucket string

// The bucket widths of a usage report.
const (
	BucketHour Bucket = "hour"
	BucketDay  Bucket = "day"
	BucketWeek Bucket = "week"
)

// UsageRequest selects usage over time, a series per tool.
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

// UsageEntry is one tool in one bucket of time, a point of a usage chart.
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

// Usage returns the error of an unknown bucket, or the runs and time per
// tool per bucket, oldest first: the series of a usage chart.
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

// bucketExpression returns the SQL naming the window of a command by the
// date or hour it starts, a label a chart plots on a time axis.
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
