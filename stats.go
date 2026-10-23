package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/atkins/chart"
	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/markdown"
	"github.com/titpetric/atkins/storage"
)

// statsLimit is how many commands the report lists: twenty fit a terminal.
const statsLimit = 20

// toolsLimit is how many tools the ranking lists.
const toolsLimit = 8

// chartHeight is the row count of the daily chart: six rows of eighths
// resolve to forty-eight steps, more than a month of days tells apart.
const chartHeight = 6

// runStats prints the stats report, or writes it as JSON or YAML.
func runStats(ctx context.Context, settings *config.Config, opts *Options) error {
	stores, err := openStorage(settings)
	if err != nil && len(stores) == 0 {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}
	defer func() {
		for _, store := range stores {
			_ = store.Close()
		}
	}()

	report, err := collectStats(ctx, stores, sampling{All: opts.All, Limit: opts.Limit})
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}

	switch {
	case opts.JSON:
		return writeJSON(os.Stdout, report)
	case opts.YAML:
		return writeYAML(os.Stdout, report)
	}

	writeStats(os.Stdout, report, isTerminal(os.Stdout))
	return nil
}

// StatsReport is everything `atkins --stats` has to say: what this
// checkout has been running, and what every checkout on the machine has
// been running between them.
//
// A run records to both databases, so the two answer different
// questions of the same rows. The project's says what standing here
// costs; the workspace's says where a week went across every project
// atkins ran in.
type StatsReport struct {
	// Project is the checkout's own history, absent for a directory
	// that belongs to no project and for one whose database is the
	// workspace's.
	Project *StatsScope `json:"project,omitempty" yaml:"project,omitempty"`

	// Workspace is every project's history together.
	Workspace *StatsScope `json:"workspace,omitempty" yaml:"workspace,omitempty"`
}

// Scopes returns the scopes to report, the project first.
func (r *StatsReport) Scopes() []*StatsScope {
	var scopes []*StatsScope
	for _, scope := range []*StatsScope{r.Project, r.Workspace} {
		if scope != nil {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

// StatsScope is one database's history, and everything a reader of
// `--stats --json` needs to draw it again elsewhere.
//
// Durations are milliseconds throughout, as the database keeps them: a
// chart takes a number and not a word.
type StatsScope struct {
	// Name names the scope, project or workspace.
	Name string `json:"name" yaml:"name"`

	// Repository is the checkout the rows came from, as its remote
	// names it; empty for the workspace, which holds many, and for a
	// project with no remote.
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`

	// Database is the file the rows came from, and Days the window
	// they cover, which is the retention window.
	Database string `json:"database" yaml:"database"`
	Days     int    `json:"days" yaml:"days"`

	// From and To bound the window, as dates.
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`

	// Runs and TotalMs are the whole window: how many commands ran and
	// how long they took between them.
	Runs    int64 `json:"runs" yaml:"runs"`
	TotalMs int64 `json:"total_ms" yaml:"total_ms"`

	// Commands are the command keys by total time.
	Commands []CommandReport `json:"commands" yaml:"commands"`

	// Tools are the tools by total time, each a roll-up of its
	// commands.
	Tools []ToolReport `json:"tools" yaml:"tools"`

	// Usage is a row per day per tool, oldest first: the series a chart
	// of tool use over time draws. A day a tool did not run has
	// no row.
	Usage []UsageReport `json:"usage" yaml:"usage"`

	// CommandUsage is a row per day per command key, oldest first: the
	// series a chart stacks by command.
	CommandUsage []CommandUsageReport `json:"command_usage" yaml:"command_usage"`

	// Samples is every command of the window one by one, oldest first,
	// with --all: the rows a reader buckets finer than a day.
	// --limit keeps the newest.
	Samples []SampleReport `json:"samples,omitempty" yaml:"samples,omitempty"`
}

// CommandReport is one command key and what it has cost. The key is
// the command's words up to its first flag, at most three and no path
// or value, as storage.Key names it: `docker compose up`, `go test`.
type CommandReport struct {
	Command   string     `json:"command" yaml:"command"`
	Tool      string     `json:"tool" yaml:"tool"`
	Runs      int64      `json:"runs" yaml:"runs"`
	TotalMs   int64      `json:"total_ms" yaml:"total_ms"`
	AverageMs int64      `json:"average_ms" yaml:"average_ms"`
	MaxMs     int64      `json:"max_ms" yaml:"max_ms"`
	Failures  int64      `json:"failures" yaml:"failures"`
	LastRunAt *time.Time `json:"last_run_at,omitempty" yaml:"last_run_at,omitempty"`
}

// ToolReport is one tool and what it has cost, across every form of it.
type ToolReport struct {
	Tool     string `json:"tool" yaml:"tool"`
	Commands int64  `json:"commands" yaml:"commands"`
	Runs     int64  `json:"runs" yaml:"runs"`
	TotalMs  int64  `json:"total_ms" yaml:"total_ms"`
	Failures int64  `json:"failures" yaml:"failures"`
}

// UsageReport is one tool on one day.
type UsageReport struct {
	Date    string `json:"date" yaml:"date"`
	Tool    string `json:"tool" yaml:"tool"`
	Runs    int64  `json:"runs" yaml:"runs"`
	TotalMs int64  `json:"total_ms" yaml:"total_ms"`
}

// CommandUsageReport is one command key on one day.
type CommandUsageReport struct {
	Date    string `json:"date" yaml:"date"`
	Command string `json:"command" yaml:"command"`
	Tool    string `json:"tool" yaml:"tool"`
	Runs    int64  `json:"runs" yaml:"runs"`
	TotalMs int64  `json:"total_ms" yaml:"total_ms"`
}

// collectStats reports each database, in the order storage.Discover returns
// them: the project's, then the workspace's. A lone database is the workspace's.
func collectStats(ctx context.Context, stores []*storage.Storage, samples sampling) (*StatsReport, error) {
	report := &StatsReport{}

	for index, store := range stores {
		name := workspaceScope
		if len(stores) > 1 && index == 0 {
			name = projectScope
		}

		scope, err := collectScope(ctx, store, name, samples)
		if err != nil {
			return nil, err
		}
		if name == projectScope {
			report.Project = scope
			continue
		}
		report.Workspace = scope
	}

	return report, nil
}

// The names a scope goes by.
const (
	projectScope   = "project"
	workspaceScope = "workspace"
)

// collectScope queries one database over the retention window.
func collectScope(ctx context.Context, store *storage.Storage, name string, samples sampling) (*StatsScope, error) {
	now := time.Now()
	since := now.Add(-storage.Retention)

	commands, err := store.Stats(ctx, storage.StatsRequest{Since: since, Limit: statsLimit})
	if err != nil {
		return nil, err
	}

	tools, err := store.Tools(ctx, storage.StatsRequest{Since: since, Limit: toolsLimit})
	if err != nil {
		return nil, err
	}

	usage, err := store.Usage(ctx, storage.UsageRequest{Since: since, Bucket: storage.BucketDay})
	if err != nil {
		return nil, err
	}

	commandUsage, err := store.CommandUsage(ctx, storage.UsageRequest{Since: since, Bucket: storage.BucketDay})
	if err != nil {
		return nil, err
	}

	report := &StatsScope{
		Name:     name,
		Database: store.Path(),
		Days:     int(storage.Retention / (24 * time.Hour)),
		From:     since.Format(time.DateOnly),
		To:       now.Format(time.DateOnly),
	}

	// A project is one checkout, so the repository its runs name is the
	// repository. A workspace holds many and names none.
	if name == projectScope {
		report.Repository = repositoryOf(ctx, store, since)
	}

	for _, command := range commands {
		report.Commands = append(report.Commands, CommandReport{
			Command:   command.Command,
			Tool:      command.Tool,
			Runs:      command.Runs,
			TotalMs:   command.TotalMs,
			AverageMs: command.AvgMs,
			MaxMs:     command.MaxMs,
			Failures:  command.Failures,
			LastRunAt: command.LastAt,
		})
	}

	// The totals come from the tools, which roll up every command and
	// not only the ones that fit the table.
	for _, tool := range tools {
		report.Tools = append(report.Tools, ToolReport(tool))
		report.Runs += tool.Runs
		report.TotalMs += tool.TotalMs
	}

	for _, entry := range usage {
		report.Usage = append(report.Usage, UsageReport{
			Date:    entry.Bucket,
			Tool:    entry.Tool,
			Runs:    entry.Runs,
			TotalMs: entry.TotalMs,
		})
	}

	for _, entry := range commandUsage {
		report.CommandUsage = append(report.CommandUsage, CommandUsageReport{
			Date:    entry.Bucket,
			Command: entry.Command,
			Tool:    entry.Tool,
			Runs:    entry.Runs,
			TotalMs: entry.TotalMs,
		})
	}

	if samples.All {
		report.Samples, err = collectSamples(ctx, store, since, samples.Limit)
		if err != nil {
			return nil, err
		}
	}

	return report, nil
}

// repositoryOf returns the repository the project's runs name, empty
// when its checkout has no remote or nothing has run in it yet.
func repositoryOf(ctx context.Context, store *storage.Storage, since time.Time) string {
	runs, err := store.Runs(ctx, storage.RunFilter{Kind: "pipeline", Since: since, Limit: 1})
	if err != nil || len(runs) == 0 {
		return ""
	}
	return runs[0].Repository
}

// writeJSON emits the report as JSON, for a program reading it.
func writeJSON(w io.Writer, report *StatsReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// writeYAML emits the report as YAML.
func writeYAML(w io.Writer, report *StatsReport) error {
	data, err := yaml.Marshal(report)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(data))
	return err
}

// writeStats writes each scope, as coloured tables or as markdown.
func writeStats(w io.Writer, report *StatsReport, color bool) {
	scopes := report.Scopes()
	if len(scopes) == 0 {
		fmt.Fprintf(w, "%s nothing recorded yet\n", colors.BrightYellow("INFO:"))
		return
	}

	for index, scope := range scopes {
		if index > 0 {
			fmt.Fprintln(w)
		}
		writeScope(w, scope, color)
	}
}

// writeScope renders one database: its header, the days, the tools and the commands.
func writeScope(w io.Writer, scope *StatsScope, color bool) {
	title := scope.Name
	if scope.Repository != "" {
		title += " · " + scope.Repository
	}
	fmt.Fprintf(w, "%s  %s\n", heading(title, color), dim(scope.Database, color))

	if len(scope.Commands) == 0 {
		fmt.Fprintf(w, "%s\n", dim("nothing recorded yet", color))
		return
	}

	fmt.Fprintf(w, "%s\n\n",
		dim(fmt.Sprintf("%d days, %s to %s, %s across %d runs",
			scope.Days, scope.From, scope.To, duration(scope.TotalMs), scope.Runs), color))

	if days := dailyChart(scope, color); days != "" {
		fmt.Fprintf(w, "%s\n", days)
	}
	if tools := toolChart(scope, color); tools != "" {
		fmt.Fprintf(w, "%s\n", tools)
	}

	fmt.Fprint(w, commandTable(scope, color))

	if len(scope.Samples) > 0 {
		fmt.Fprintln(w)
		fmt.Fprint(w, sampleTable(scope, color))
	}
}

// dailyChart draws a column per day of the window.
func dailyChart(report *StatsScope, color bool) string {
	bars := dailyBars(report)
	if len(bars) == 0 {
		return ""
	}

	return chart.Vertical(chart.VerticalOptions{
		Bars:   bars,
		Height: chartHeight,
		Format: duration,
		Color:  color,
	})
}

// dailyBars folds the usage rows into one bar per day, empty days included.
func dailyBars(report *StatsScope) []chart.Bar {
	from, err := time.Parse(time.DateOnly, report.From)
	if err != nil {
		return nil
	}
	to, err := time.Parse(time.DateOnly, report.To)
	if err != nil {
		return nil
	}

	totals := map[string]int64{}
	for _, entry := range report.Usage {
		totals[entry.Date] += entry.TotalMs
	}

	var bars []chart.Bar
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		date := day.Format(time.DateOnly)
		bars = append(bars, chart.Bar{Label: date, Value: totals[date]})
	}

	return bars
}

// toolChart ranks the tools by the time they have taken.
func toolChart(report *StatsScope, color bool) string {
	if len(report.Tools) < 2 {
		return ""
	}

	bars := make([]chart.Bar, 0, len(report.Tools))
	runs := map[string]int64{}
	for _, tool := range report.Tools {
		bars = append(bars, chart.Bar{Label: tool.Tool, Value: tool.TotalMs})
		runs[tool.Tool] = tool.Runs
	}

	return chart.Horizontal(chart.HorizontalOptions{
		Bars:   bars,
		Format: duration,
		Note: func(bar chart.Bar) string {
			return plural(runs[bar.Label], "run")
		},
		Color: color,
	})
}

// commandTable renders the commands by total time.
func commandTable(report *StatsScope, color bool) string {
	rows := []string{
		"| Command | Runs | Total | Avg | Max | Failed | Last |",
		"| --- | --- | --- | --- | --- | --- | --- |",
	}
	for _, command := range report.Commands {
		rows = append(rows, "| "+strings.Join([]string{
			command.Command,
			strconv.FormatInt(command.Runs, 10),
			duration(command.TotalMs),
			duration(command.AverageMs),
			duration(command.MaxMs),
			strconv.FormatInt(command.Failures, 10),
			ago(command.LastRunAt),
		}, " | ")+" |")
	}

	table := strings.Join(rows, "\n") + "\n"
	if !color {
		return table
	}
	return markdown.Render(table, true) + "\n"
}

// heading renders the path the rows came from.
func heading(text string, color bool) string {
	if !color {
		return text
	}
	return colors.BrightWhite(text)
}

// dim renders a line a reader skims past.
func dim(text string, color bool) string {
	if !color {
		return text
	}
	return colors.Dim(text)
}

// plural renders a count with its noun.
func plural(n int64, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.FormatInt(n, 10) + " " + noun + "s"
}

// duration renders milliseconds the way a person reads a duration.
func duration(ms int64) string {
	switch {
	case ms <= 0:
		return "0s"
	case ms < 1000:
		return strconv.FormatInt(ms, 10) + "ms"
	case ms < 60_000:
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	case ms < 3_600_000:
		return fmt.Sprintf("%dm%02ds", ms/60_000, (ms%60_000)/1000)
	default:
		return fmt.Sprintf("%dh%02dm", ms/3_600_000, (ms%3_600_000)/60_000)
	}
}

// ago renders how long ago a time is, and a dash for no time.
func ago(at *time.Time) string {
	if at == nil {
		return "-"
	}

	since := time.Since(*at)
	switch {
	case since < time.Minute:
		return "just now"
	case since < time.Hour:
		return fmt.Sprintf("%dm ago", int(since.Minutes()))
	case since < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(since.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(since.Hours()/24))
	}
}
