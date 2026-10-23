package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestWriteStatsTable(t *testing.T) {
	at := time.Now().Add(-3 * time.Hour)
	stats := []storage.CommandStat{
		{Normalized: "go test -cover", Tool: "go", Runs: 42, TotalMs: 184_000, AvgMs: 4_380, MaxMs: 9_100, Failures: 3, LastAt: &at},
		{Normalized: "npm run build", Tool: "npm", Runs: 2, TotalMs: 900, AvgMs: 450, MaxMs: 500},
	}

	var out bytes.Buffer
	writeStats(&out, "/w/.atkins/atkins.db", stats, false)
	table := out.String()

	// Not a terminal, so plain markdown and no escape codes: the same
	// command serves a terminal and a document.
	assert.NotContains(t, table, "\x1b[")
	assert.Contains(t, table, "| Command | Runs |")
	assert.Contains(t, table, "go test -cover")
	assert.Contains(t, table, "3m04s")
	assert.Contains(t, table, "4.4s")
	assert.Contains(t, table, "9.1s")
	assert.Contains(t, table, "3h ago")
	// A command that has never failed still shows a count, so the column
	// never reads as missing data.
	assert.Contains(t, table, "900ms")
	assert.Contains(t, table, "| - |")
}

func TestWriteStatsOnATerminal(t *testing.T) {
	at := time.Now().Add(-3 * time.Hour)
	stats := []storage.CommandStat{
		{Normalized: "go test -cover", Tool: "go", Runs: 42, TotalMs: 184_000, AvgMs: 4_380, MaxMs: 9_100, Failures: 3, LastAt: &at},
	}

	var out bytes.Buffer
	writeStats(&out, "/w/.atkins/atkins.db", stats, true)
	table := out.String()

	// A terminal gets the bordered, coloured grid the help document
	// draws its tables with.
	assert.Contains(t, table, "\x1b[")
	assert.Contains(t, table, "┌")
	assert.Contains(t, table, "go test -cover")
	assert.NotContains(t, table, "| Command | Runs |")
}

func TestWriteStatsAsJSONAndYAML(t *testing.T) {
	at := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	stats := []storage.CommandStat{
		{Normalized: "go test -cover", Tool: "go", Runs: 42, TotalMs: 184_000, AvgMs: 4_380, MaxMs: 9_100, Failures: 3, LastAt: &at},
	}

	// The machine-readable output carries milliseconds, not words: a
	// reader charting them wants a number.
	var asJSON bytes.Buffer
	assert.NoError(t, writeJSON(&asJSON, stats))
	assert.Contains(t, asJSON.String(), `"command": "go test -cover"`)
	assert.Contains(t, asJSON.String(), `"total_ms": 184000`)
	assert.Contains(t, asJSON.String(), `"last_run_at": "2026-10-23T09:00:00Z"`)
	assert.NotContains(t, asJSON.String(), "3m04s")

	var asYAML bytes.Buffer
	assert.NoError(t, writeYAML(&asYAML, stats))
	assert.Contains(t, asYAML.String(), "command: go test -cover")
	assert.Contains(t, asYAML.String(), "total_ms: 184000")
}

func TestWriteStatsOnNothingRecorded(t *testing.T) {
	var out bytes.Buffer
	writeStats(&out, "/w/.atkins/atkins.db", nil, true)

	// The path is in the message: the usual cause of an empty report is
	// a database somewhere other than where the reader expected.
	assert.Contains(t, out.String(), "/w/.atkins/atkins.db")
	assert.Contains(t, out.String(), "nothing recorded")
}

func TestDuration(t *testing.T) {
	tests := map[int64]string{
		0:          "0s",
		-1:         "0s",
		12:         "12ms",
		999:        "999ms",
		1_000:      "1.0s",
		59_999:     "60.0s",
		60_000:     "1m00s",
		184_000:    "3m04s",
		3_600_000:  "1h00m",
		7_500_000:  "2h05m",
		90_061_000: "25h01m",
	}

	for ms, want := range tests {
		assert.Equal(t, want, duration(ms))
	}
}

func TestAgo(t *testing.T) {
	assert.Equal(t, "-", ago(nil))

	for since, want := range map[time.Duration]string{
		time.Second:         "just now",
		90 * time.Second:    "1m ago",
		3 * time.Hour:       "3h ago",
		50 * time.Hour:      "2d ago",
		30 * 24 * time.Hour: "30d ago",
	} {
		at := time.Now().Add(-since)
		assert.Equal(t, want, ago(&at))
	}
}
