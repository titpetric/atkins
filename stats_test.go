package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/eventlog"
	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

// sample is a report of two tools over three days, enough for every
// part of the rendering to have something to draw.
func sample() *StatsReport {
	return &StatsReport{Workspace: sampleScope()}
}

// sampleScope is one database of two tools over three days.
func sampleScope() *StatsScope {
	at := time.Now().Add(-3 * time.Hour)
	return &StatsScope{
		Name:     "workspace",
		Database: "/w/.atkins/atkins.db",
		Days:     31,
		From:     "2026-10-21",
		To:       "2026-10-23",
		Runs:     44,
		TotalMs:  184_900,
		Commands: []CommandReport{
			{Command: "go test -cover", Tool: "go", Runs: 42, TotalMs: 184_000, AverageMs: 4_380, MaxMs: 9_100, Failures: 3, LastRunAt: &at},
			{Command: "npm run build", Tool: "npm", Runs: 2, TotalMs: 900, AverageMs: 450, MaxMs: 500},
		},
		Tools: []ToolReport{
			{Tool: "go", Commands: 1, Runs: 42, TotalMs: 184_000, Failures: 3},
			{Tool: "npm", Commands: 1, Runs: 2, TotalMs: 900},
		},
		Usage: []UsageReport{
			{Date: "2026-10-21", Tool: "go", Runs: 20, TotalMs: 90_000},
			{Date: "2026-10-23", Tool: "go", Runs: 22, TotalMs: 94_000},
			{Date: "2026-10-23", Tool: "npm", Runs: 2, TotalMs: 900},
		},
	}
}

func TestWriteStatsPlain(t *testing.T) {
	var out bytes.Buffer
	writeStats(&out, sample(), false)
	text := out.String()

	// The writer is a buffer, and the output is plain markdown with no
	// escape codes: the same command serves a terminal and a document.
	assert.NotContains(t, text, "\x1b[")
	assert.Contains(t, text, "/w/.atkins/atkins.db")
	assert.Contains(t, text, "31 days, 2026-10-21 to 2026-10-23, 3m04s across 44 runs")

	// The scope names itself and the file its rows came from.
	assert.Contains(t, text, "workspace")

	// The chart of the days, the ranking of the tools, then the table.
	assert.Contains(t, text, "█")
	assert.Contains(t, text, "2026-10-21")
	assert.Contains(t, text, "| Command | Runs |")
	assert.Contains(t, text, "go test -cover")
	assert.Contains(t, text, "3h ago")
	assert.Contains(t, text, "| - |")

	// The tool ranking carries the run counts beside the bars.
	assert.Contains(t, text, "42 runs")
	assert.Contains(t, text, "2 runs")
}

func TestWriteStatsOnATerminal(t *testing.T) {
	var out bytes.Buffer
	writeStats(&out, sample(), true)
	text := out.String()

	// A terminal gets the bordered, coloured grid the help document
	// draws its tables with.
	assert.Contains(t, text, "\x1b[")
	assert.Contains(t, text, "┌")
	assert.Contains(t, text, "go test -cover")
	assert.NotContains(t, text, "| Command | Runs |")
}

func TestWriteStatsAsJSONAndYAML(t *testing.T) {
	report := sample()

	// The machine-readable output carries milliseconds as numbers, and
	// the per-day series a chart of tool use over time draws.
	var asJSON bytes.Buffer
	assert.NoError(t, writeJSON(&asJSON, report))
	assert.Contains(t, asJSON.String(), `"command": "go test -cover"`)
	assert.Contains(t, asJSON.String(), `"total_ms": 184000`)
	assert.Contains(t, asJSON.String(), `"days": 31`)
	assert.Contains(t, asJSON.String(), `"date": "2026-10-23"`)
	assert.Contains(t, asJSON.String(), `"tool": "npm"`)
	assert.NotContains(t, asJSON.String(), "3m04s")

	var asYAML bytes.Buffer
	assert.NoError(t, writeYAML(&asYAML, report))
	assert.Contains(t, asYAML.String(), "command: go test -cover")
	assert.Contains(t, asYAML.String(), "total_ms: 184000")
	assert.Contains(t, asYAML.String(), "date: \"2026-10-21\"")
}

func TestWriteStatsOnNothingRecorded(t *testing.T) {
	var out bytes.Buffer
	writeStats(&out, &StatsReport{}, true)

	assert.Contains(t, out.String(), "nothing recorded")
}

// A day nothing ran is a bar of zero. A run of days reads as time.
func TestDailyBarsCoverEveryDay(t *testing.T) {
	bars := dailyBars(sampleScope())

	assert.Len(t, bars, 3)
	assert.Equal(t, "2026-10-21", bars[0].Label)
	assert.Equal(t, int64(90_000), bars[0].Value)
	assert.Equal(t, "2026-10-22", bars[1].Label)
	assert.Equal(t, int64(0), bars[1].Value)
	// Both tools of a day fold into its one bar.
	assert.Equal(t, int64(94_900), bars[2].Value)
}

func TestDailyBarsWithoutADate(t *testing.T) {
	assert.Nil(t, dailyBars(&StatsScope{From: "not-a-date", To: "2026-10-23"}))
	assert.Nil(t, dailyBars(&StatsScope{From: "2026-10-21", To: "not-a-date"}))
}

// One tool is a ranking of one, which says nothing a table does not.
func TestToolChartNeedsTwo(t *testing.T) {
	scope := sampleScope()
	scope.Tools = scope.Tools[:1]

	assert.Equal(t, "", toolChart(scope, false))
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

// The plain report is a document: every line reads as text with no
// control characters.
func TestPlainReportIsPlain(t *testing.T) {
	var out bytes.Buffer
	writeStats(&out, sample(), false)

	for _, line := range strings.Split(out.String(), "\n") {
		assert.NotContains(t, line, "\x1b")
	}
}

// TestStatsReport_Scopes covers the order of the scopes: the project
// first, the context the reader stands in.
func TestStatsReport_Scopes(t *testing.T) {
	project := sampleScope()
	project.Name = "project"
	workspace := sampleScope()

	assert.Len(t, (&StatsReport{}).Scopes(), 0)

	only := (&StatsReport{Workspace: workspace}).Scopes()
	assert.Len(t, only, 1)
	assert.Equal(t, "workspace", only[0].Name)

	both := (&StatsReport{Project: project, Workspace: workspace}).Scopes()
	assert.Len(t, both, 2)
	assert.Equal(t, "project", both[0].Name)
	assert.Equal(t, "workspace", both[1].Name)
}

// Both scopes render, each with its own heading, chart, ranking and
// table.
func TestWriteStatsBothScopes(t *testing.T) {
	project := sampleScope()
	project.Name = "project"
	project.Repository = "titpetric/atkins"
	project.Database = "/w/atkins/.atkins/atkins.db"

	var out bytes.Buffer
	writeStats(&out, &StatsReport{Project: project, Workspace: sampleScope()}, false)
	text := out.String()

	assert.Contains(t, text, "project · titpetric/atkins")
	assert.Contains(t, text, "/w/atkins/.atkins/atkins.db")
	assert.Contains(t, text, "workspace")
	assert.Contains(t, text, "/w/.atkins/atkins.db")

	// One of everything per scope.
	assert.Equal(t, 2, strings.Count(text, "| Command | Runs |"))
	assert.Equal(t, 2, strings.Count(text, "31 days, 2026-10-21 to 2026-10-23"))
}

// A scope whose database holds nothing says so under its own heading,
// and the other scope keeps its own.
func TestWriteStatsOneEmptyScope(t *testing.T) {
	project := &StatsScope{Name: "project", Database: "/w/atkins/.atkins/atkins.db"}

	var out bytes.Buffer
	writeStats(&out, &StatsReport{Project: project, Workspace: sampleScope()}, false)
	text := out.String()

	assert.Contains(t, text, "/w/atkins/.atkins/atkins.db")
	assert.Contains(t, text, "nothing recorded yet")
	assert.Equal(t, 1, strings.Count(text, "| Command | Runs |"))
}

// The JSON carries the two scopes under their own keys, and leaves out
// the project a directory outside one has none of.
func TestWriteStatsJSONScopes(t *testing.T) {
	project := sampleScope()
	project.Name = "project"
	project.Repository = "titpetric/atkins"

	var both bytes.Buffer
	assert.NoError(t, writeJSON(&both, &StatsReport{Project: project, Workspace: sampleScope()}))
	assert.Contains(t, both.String(), `"project": {`)
	assert.Contains(t, both.String(), `"workspace": {`)
	assert.Contains(t, both.String(), `"repository": "titpetric/atkins"`)

	var alone bytes.Buffer
	assert.NoError(t, writeJSON(&alone, &StatsReport{Workspace: sampleScope()}))
	assert.NotContains(t, alone.String(), `"project"`)
	assert.Contains(t, alone.String(), `"workspace": {`)
}

func TestWriteStatsSamples(t *testing.T) {
	report := sample()

	// Without --all the report carries no samples key at all.
	var without bytes.Buffer
	assert.NoError(t, writeJSON(&without, report))
	assert.NotContains(t, without.String(), `"samples"`)

	report.Workspace.Samples = []SampleReport{
		{At: "2026-10-23T09:15:02.118Z", Command: "go test", Tool: "go", Kind: "step", DurationMs: 4_200, Job: "test", InvocationID: "01HRUN"},
		{At: "2026-10-23T09:16:40.000Z", Command: "npm run build", Tool: "npm", Kind: "step", DurationMs: 450, ExitCode: 2, Failed: true, InvocationID: "01HRUN"},
	}

	var with bytes.Buffer
	assert.NoError(t, writeJSON(&with, report))
	assert.Contains(t, with.String(), `"samples"`)
	assert.Contains(t, with.String(), `"at": "2026-10-23T09:15:02.118Z"`)
	assert.Contains(t, with.String(), `"failed": true`)

	var text bytes.Buffer
	writeStats(&text, report, false)
	assert.Contains(t, text.String(), "| At | Command | Job | Took | Exit |")
	assert.Contains(t, text.String(), "| go test | test | 4.2s | 0 |")
	assert.Contains(t, text.String(), "| npm run build | - | 450ms | 2 |")
}

func TestCollectStatsSamples(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewStorageAt(ctx, storage.Memory)
	assert.NoError(t, err)
	defer store.Close()

	at := time.Now().Add(-time.Hour).UTC()
	_, err = store.Record(ctx, storage.RecordRequest{Log: &eventlog.Log{
		Metadata: eventlog.RunMetadata{CreatedAt: at, Pipeline: "Demo"},
		State: &eventlog.StateNode{Name: "Demo", Children: []*eventlog.StateNode{{
			Name: "build", ID: "jobs.build",
			Children: []*eventlog.StateNode{
				{Name: "go build", ID: "jobs.build.steps.0", Duration: 2},
				{Name: "go vet", ID: "jobs.build.steps.1", Start: 2, Duration: 1},
			},
		}}},
		Events: []*eventlog.Event{
			{ID: "jobs.build.steps.0", Type: eventlog.EventTypeStep, Command: "go build -ldflags=-X=main.token=abc .", Duration: 2},
			{ID: "jobs.build.steps.1", Type: eventlog.EventTypeStep, Command: "go vet ./...", Start: 2, Duration: 1, ExitCode: 1},
		},
	}})
	assert.NoError(t, err)

	report, err := collectStats(ctx, []*storage.Storage{store}, sampling{})
	assert.NoError(t, err)
	assert.Len(t, report.Workspace.Samples, 0)

	report, err = collectStats(ctx, []*storage.Storage{store}, sampling{All: true})
	assert.NoError(t, err)
	samples := report.Workspace.Samples
	assert.Len(t, samples, 2)
	assert.Equal(t, at.Truncate(time.Millisecond).Format(sampleTime), samples[0].At)
	assert.Equal(t, "go build", samples[0].Command)
	assert.Equal(t, "build", samples[0].Job)
	assert.Equal(t, "Demo", samples[0].Pipeline)
	assert.Equal(t, "go vet", samples[1].Command)
	assert.True(t, samples[1].Failed)

	// The key and never the command line: the value given is not there.
	var out bytes.Buffer
	assert.NoError(t, writeJSON(&out, report))
	assert.NotContains(t, out.String(), "abc")

	report, err = collectStats(ctx, []*storage.Storage{store}, sampling{All: true, Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, report.Workspace.Samples, 1)
	assert.Equal(t, "go vet", report.Workspace.Samples[0].Command)
}

// A mirror in the database setting takes the rows of a run and nothing
// else: the report opens the project's and the workspace's alone. Its
// output stays the same with or without one.
func TestOpenStatsStorageSkipsMirrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(storage.EnvDB, "")

	project := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(project, ".atkins"), 0o755))
	t.Chdir(project)

	workspace := filepath.Join(t.TempDir(), "ws", "atkins.db")
	mirror := filepath.Join(t.TempDir(), "instance", "atkins.db")
	settings, err := config.Default()
	assert.NoError(t, err)
	settings.Storage.Database = workspace + string(os.PathListSeparator) + mirror

	recorders, err := openStorage(settings)
	assert.NoError(t, err)
	assert.Len(t, recorders, 3)
	for _, store := range recorders {
		assert.NoError(t, store.Close())
	}

	readers, err := openStatsStorage(settings)
	assert.NoError(t, err)
	assert.Len(t, readers, 2)
	assert.Equal(t, filepath.Join(project, ".atkins", "atkins.db"), readers[0].Path())
	assert.Equal(t, workspace, readers[1].Path())
	for _, store := range readers {
		assert.NoError(t, store.Close())
	}
}
