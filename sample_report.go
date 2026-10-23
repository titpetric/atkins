package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/titpetric/atkins/markdown"
	"github.com/titpetric/atkins/storage"
)

// SampleReport is one recorded command. It names the command by its
// key and never by its command line, because a command line carries
// its values and a value can be a token.
type SampleReport struct {
	// At is when the command started, in UTC to the millisecond.
	At string `json:"at" yaml:"at"`

	Command    string `json:"command" yaml:"command"`
	Tool       string `json:"tool" yaml:"tool"`
	Kind       string `json:"kind" yaml:"kind"`
	DurationMs int64  `json:"duration_ms" yaml:"duration_ms"`
	ExitCode   int64  `json:"exit_code" yaml:"exit_code"`
	Failed     bool   `json:"failed" yaml:"failed"`

	// Pipeline and Job are what ran it, and InvocationID the atkins
	// process, shared by every command of one run.
	Pipeline     string `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Job          string `json:"job,omitempty" yaml:"job,omitempty"`
	InvocationID string `json:"invocation_id" yaml:"invocation_id"`
}

// sampleTime is how a sample spells its time: RFC 3339 in UTC, to the
// millisecond.
const sampleTime = "2006-01-02T15:04:05.000Z07:00"

// sampling is whether a report lists its commands one by one, and how
// many of the newest it keeps. A zero Limit keeps them all.
type sampling struct {
	All   bool
	Limit int
}

// collectSamples lists the commands of the window one by one, oldest
// first.
func collectSamples(ctx context.Context, store *storage.Storage, since time.Time, limit int) ([]SampleReport, error) {
	rows, err := store.Samples(ctx, storage.SampleRequest{Since: since, Limit: limit})
	if err != nil {
		return nil, err
	}

	samples := make([]SampleReport, 0, len(rows))
	for _, row := range rows {
		samples = append(samples, SampleReport{
			At:           row.At.UTC().Format(sampleTime),
			Command:      row.Command,
			Tool:         row.Tool,
			Kind:         row.Kind,
			DurationMs:   row.DurationMs,
			ExitCode:     row.ExitCode,
			Failed:       row.ExitCode != 0,
			Pipeline:     row.Pipeline,
			Job:          row.Job,
			InvocationID: row.InvocationID,
		})
	}

	return samples, nil
}

// sampleTable lists the commands one by one, oldest first, as --all
// asks: when, what, how long and how it ended.
func sampleTable(report *StatsScope, color bool) string {
	rows := []string{
		"| At | Command | Job | Took | Exit |",
		"| --- | --- | --- | --- | --- |",
	}
	for _, sample := range report.Samples {
		at := sample.At
		if parsed, err := time.Parse(sampleTime, sample.At); err == nil {
			at = parsed.Local().Format(time.DateTime)
		}
		job := sample.Job
		if job == "" {
			job = "-"
		}
		rows = append(rows, "| "+strings.Join([]string{
			at,
			sample.Command,
			job,
			duration(sample.DurationMs),
			strconv.FormatInt(sample.ExitCode, 10),
		}, " | ")+" |")
	}

	table := strings.Join(rows, "\n") + "\n"
	if !color {
		return table
	}
	return markdown.Render(table, true) + "\n"
}
