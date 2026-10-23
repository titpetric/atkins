package storage

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/model"
)

// RecordRequest is a finished run to store.
type RecordRequest struct {
	// Log is the event log document the run ended with: its metadata,
	// the tree it walked, every event it collected and the summary.
	Log *eventlog.Log

	// Dir is the directory atkins ran in. Empty records none.
	Dir string
}

// RecordResult says what one Record wrote.
type RecordResult struct {
	// InvocationID is the run row of the pipeline node, the value every
	// row of this invocation carries.
	InvocationID string

	// Runs and Commands are how many rows of each kind were written.
	Runs     int
	Commands int
}

// Record stores a finished run: a run row per node of the tree and a
// command row per command the run executed.
//
// The rows go in one transaction. A run is read as a whole - the job's
// time against its steps' - and half of one recorded is worse than none,
// because a report would quietly treat it as complete.
func (s *Storage) Record(ctx context.Context, req RecordRequest) (*RecordResult, error) {
	if s == nil || req.Log == nil || req.Log.State == nil {
		return &RecordResult{}, nil
	}

	rows, ids := runRows(req)
	commands := commandRows(req, rows[0].InvocationID, ids)

	client := s.client()
	defer client.Close()

	if err := client.Begin(ctx); err != nil {
		return nil, err
	}

	for _, row := range rows {
		if err := client.Insert(ctx, model.RunTable, row); err != nil {
			_ = client.Rollback()
			return nil, err
		}
	}
	for _, row := range commands {
		if err := client.Insert(ctx, model.CommandTable, row); err != nil {
			_ = client.Rollback()
			return nil, err
		}
	}

	if err := client.Commit(); err != nil {
		return nil, err
	}

	return &RecordResult{
		InvocationID: rows[0].InvocationID,
		Runs:         len(rows),
		Commands:     len(commands),
	}, nil
}

// runRows flattens the state tree into run rows, and returns them with a
// map from the node id the events name to the row id they belong to.
//
// The root is the pipeline node and carries the metadata of the whole
// invocation: the file, the directory, the module and the git state. Its
// children are jobs and everything below them is a step, which is the
// shape the tree has: a pipeline holds jobs and a job holds steps.
func runRows(req RecordRequest) ([]*model.Run, map[string]string) {
	var (
		rows = make([]*model.Run, 0, 8)
		ids  = map[string]string{}
		meta = req.Log.Metadata
		base = meta.CreatedAt
	)

	var walk func(node *eventlog.StateNode, parentID string, depth int)
	walk = func(node *eventlog.StateNode, parentID string, depth int) {
		if node == nil {
			return
		}

		row := &model.Run{
			ID:         newID(),
			ParentID:   parentID,
			Kind:       kindAt(depth),
			Name:       name(node, depth),
			Result:     result(node.Result),
			Steps:      int64(node.Steps),
			DurationMs: milliseconds(node.Duration),
			StartedAt:  stamp(base.Add(seconds(node.Start))),
			CreatedAt:  stamp(time.Now()),
		}
		row.FinishedAt = stamp(base.Add(seconds(node.Start + node.Duration)))

		if depth == 0 {
			row.Pipeline = meta.Pipeline
			row.File = meta.File
			row.Dir = req.Dir
			row.Module = meta.ModulePath
			if meta.Git != nil {
				row.Repository = meta.Git.Repository
				row.Branch = meta.Git.Branch
				row.CommitSha = meta.Git.Commit
			}
			if req.Log.Summary != nil {
				row.Steps = int64(req.Log.Summary.TotalSteps)
				row.DurationMs = milliseconds(req.Log.Summary.Duration)
				row.Result = result(req.Log.Summary.Result)
				row.FinishedAt = stamp(base.Add(seconds(req.Log.Summary.Duration)))
			}
		}

		rows = append(rows, row)
		if node.ID != "" {
			ids[node.ID] = row.ID
		}
		// A job is addressed by name too: a $() substitution outside any
		// step records the job name as its parent.
		if depth == 1 && node.Name != "" {
			ids[node.Name] = row.ID
		}

		for _, child := range node.Children {
			walk(child, row.ID, depth+1)
		}
	}

	walk(req.Log.State, "", 0)

	invocation := rows[0].ID
	for _, row := range rows {
		row.InvocationID = invocation
	}

	return rows, ids
}

// commandRows turns the events that ran something into command rows. An
// event with no command is a step's outcome, which the run rows already
// carry.
func commandRows(req RecordRequest, invocation string, ids map[string]string) []*model.Command {
	var (
		rows = make([]*model.Command, 0, len(req.Log.Events))
		base = req.Log.Metadata.CreatedAt
		now  = stamp(time.Now())
	)

	for _, event := range req.Log.Events {
		command := strings.TrimSpace(event.Command)
		if command == "" {
			continue
		}

		normalized, tool := Normalize(command)
		// A builtin is the shell setting itself up, and `set -eu` costs
		// nothing worth a row.
		if IsBuiltin(tool) {
			continue
		}
		parent := event.ParentID
		if parent == "" {
			parent = event.ID
		}

		rows = append(rows, &model.Command{
			ID:           newID(),
			InvocationID: invocation,
			ParentID:     ids[parent],
			Kind:         commandKind(event.Type),
			Command:      command,
			Normalized:   normalized,
			Tool:         tool,
			Dir:          event.Dir,
			ExitCode:     int64(event.ExitCode),
			Error:        event.Error,
			DurationMs:   milliseconds(event.Duration),
			StartedAt:    stamp(base.Add(seconds(event.Start))),
			CreatedAt:    now,
		})
	}

	return rows
}

// name is what the row is grouped by over time.
//
// A job's displayed name carries its description, which an author edits
// freely, so a job row takes the canonical name out of the node's id
// instead: `jobs.release` is release, this month and next. A step's
// displayed name is the command it ran, which is what a reader of a
// single run wants to see, so a step keeps it.
func name(node *eventlog.StateNode, depth int) string {
	if depth == 1 && strings.HasPrefix(node.ID, jobIDPrefix) {
		return strings.TrimPrefix(node.ID, jobIDPrefix)
	}
	return node.Name
}

// jobIDPrefix is how the runner spells a job node's id.
const jobIDPrefix = "jobs."

// kindAt names the node at a depth of the tree.
func kindAt(depth int) string {
	switch depth {
	case 0:
		return "pipeline"
	case 1:
		return "job"
	default:
		return "step"
	}
}

// result maps an event log result onto the column, which allows three
// values and defaults to pass.
func result(value eventlog.Result) string {
	switch value {
	case eventlog.ResultFail:
		return "fail"
	case eventlog.ResultSkipped:
		return "skipped"
	default:
		return "pass"
	}
}

// commandKind maps an event type onto the command kind.
func commandKind(value eventlog.EventType) string {
	switch value {
	case eventlog.EventTypeSubstitution:
		return "substitution"
	case eventlog.EventTypeInterpolation:
		return "interpolation"
	default:
		return "step"
	}
}

// seconds turns a duration in float seconds into a time.Duration.
func seconds(value float64) time.Duration {
	return time.Duration(value * float64(time.Second))
}

// milliseconds turns a duration in float seconds into whole
// milliseconds, which is the resolution the columns keep.
//
// Rounded, not truncated: truncating loses half a millisecond on every
// row, and a report that sums thousands of them is then short by
// seconds it cannot account for.
func milliseconds(value float64) int64 {
	if value <= 0 {
		return 0
	}
	return int64(math.Round(value * 1000))
}
