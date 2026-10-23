package storage

import (
	"context"
)

// sampleRow is one command row as the query returns it. started_at
// arrives as text through the COALESCE, and Samples parses it.
type sampleRow struct {
	StartedAt    string `db:"started_at"`
	Normalized   string `db:"normalized"`
	Kind         string `db:"kind"`
	DurationMs   int64  `db:"duration_ms"`
	ExitCode     int64  `db:"exit_code"`
	Pipeline     string `db:"pipeline"`
	Job          string `db:"job"`
	InvocationID string `db:"invocation_id"`
}

// Samples returns the commands of the request one by one, oldest first.
//
// The filter runs on the command table alone, in a subquery, because
// the run rows it joins share its column names.
//
// A command's job is the job node above its step, or the node it names
// itself when a $() substitution ran outside any step.
func (s *Storage) Samples(ctx context.Context, req SampleRequest) ([]Sample, error) {
	where, args := commandWhere(req.Since, req.Until, req.Tool, req.Dir)

	query := `SELECT
			COALESCE(command.started_at, '') AS started_at,
			command.normalized,
			command.kind,
			command.duration_ms,
			command.exit_code,
			COALESCE(invocation.pipeline, '') AS pipeline,
			CASE
				WHEN parent.kind = 'job' THEN parent.name
				WHEN grandparent.kind = 'job' THEN grandparent.name
				ELSE ''
			END AS job,
			command.invocation_id
		FROM (SELECT * FROM command` + where + `) AS command
		LEFT JOIN run AS invocation ON invocation.id = command.invocation_id
		LEFT JOIN run AS parent ON parent.id = command.parent_id
		LEFT JOIN run AS grandparent ON grandparent.id = parent.parent_id
		ORDER BY command.started_at DESC, command.id DESC`

	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	}

	client := s.client()
	defer client.Close()

	rows, err := client.Select[sampleRow](ctx, query, args...)
	if err != nil {
		return nil, err
	}

	samples := make([]Sample, 0, len(rows))
	// The query reads newest first so a limit keeps the newest; the
	// report is oldest first, the order a chart draws a series in.
	for index := len(rows) - 1; index >= 0; index-- {
		row := rows[index]
		key, tool := Key(row.Normalized)
		if key == "" {
			continue
		}
		sample := Sample{
			Command:      key,
			Tool:         tool,
			Kind:         row.Kind,
			DurationMs:   row.DurationMs,
			ExitCode:     row.ExitCode,
			Pipeline:     row.Pipeline,
			Job:          row.Job,
			InvocationID: row.InvocationID,
		}
		if at := parseStamp(row.StartedAt); at != nil {
			sample.At = *at
		}
		samples = append(samples, sample)
	}

	return samples, nil
}
