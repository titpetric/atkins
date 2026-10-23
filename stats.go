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

// statsLimit is how many commands the report lists. Twenty fits a
// terminal, and the tail of a long tail is not what anyone is looking
// for.
const statsLimit = 20

// toolsLimit is how many tools the ranking lists. A checkout that uses
// more than eight is not a checkout whose ninth is worth a row.
const toolsLimit = 8

// chartHeight is how many rows the daily chart is drawn in. Six rows of
// eighths resolve to forty-eight steps, which is more than a month of
// days can show apart anyway.
const chartHeight = 6

// runStats reports what has been recorded: the days, the tools, and the
// commands by the time they have taken.
//
// The question it answers is where the time goes. Not which command is
// slowest, but which one costs the most once its frequency is counted,
// and whether that has been true all month or only since Tuesday.
func runStats(ctx context.Context, settings *config.Config, opts *Options) error {
	store, err := openStorage(settings)
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}
	defer store.Close()

	report, err := collectStats(ctx, store)
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

// StatsReport is everything `atkins --stats` has to say, and everything
// a reader of `--stats --json` needs to draw it again elsewhere.
//
// Durations are milliseconds throughout, as the database keeps them,
// because a reader charting them wants a number and not a word.
type StatsReport struct {
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

	// Commands are the normalized commands by total time.
	Commands []CommandReport `json:"commands" yaml:"commands"`

	// Tools are the tools by total time, each a roll-up of its
	// commands.
	Tools []ToolReport `json:"tools" yaml:"tools"`

	// Usage is a row per day per tool, oldest first: the series a chart
	// of tool use over time is drawn from. A day a tool did not run has
	// no row.
	Usage []UsageReport `json:"usage" yaml:"usage"`
}

// CommandReport is one normalized command and what it has cost.
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

// collectStats asks the storage its three questions over the retention
// window and assembles the report.
func collectStats(ctx context.Context, store *storage.Storage) (*StatsReport, error) {
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

	report := &StatsReport{
		Database: store.Path(),
		Days:     int(storage.Retention / (24 * time.Hour)),
		From:     since.Format(time.DateOnly),
		To:       now.Format(time.DateOnly),
	}

	for _, command := range commands {
		report.Commands = append(report.Commands, CommandReport{
			Command:   command.Normalized,
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

	return report, nil
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

// writeStats renders the report for a person: the days, the tools, then
// the commands.
//
// On a terminal the table is drawn with borders and colour, the way
// `atkins --help` draws its tables. Redirected to a file or a pipe it
// is plain markdown, so the same command serves a terminal and a
// document without a flag to say which.
func writeStats(w io.Writer, report *StatsReport, color bool) {
	if len(report.Commands) == 0 {
		fmt.Fprintf(w, "%s nothing recorded yet in %s\n", colors.BrightYellow("INFO:"), report.Database)
		return
	}

	fmt.Fprintf(w, "%s\n", heading(report.Database, color))
	fmt.Fprintf(w, "%s\n\n",
		dim(fmt.Sprintf("%d days, %s to %s, %s across %d runs",
			report.Days, report.From, report.To, duration(report.TotalMs), report.Runs), color))

	if days := dailyChart(report, color); days != "" {
		fmt.Fprintf(w, "%s\n", days)
	}
	if tools := toolChart(report, color); tools != "" {
		fmt.Fprintf(w, "%s\n", tools)
	}

	fmt.Fprint(w, commandTable(report, color))
}

// dailyChart draws a column per day of the window, so a reader sees
// whether the month was steady or was one Tuesday.
func dailyChart(report *StatsReport, color bool) string {
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

// dailyBars folds the per-tool usage rows into one bar per day of the
// window, including the days nothing ran: a run of days reads as time
// only when the empty ones are drawn.
func dailyBars(report *StatsReport) []chart.Bar {
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
func toolChart(report *StatsReport, color bool) string {
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
			return fmt.Sprintf("%d runs", runs[bar.Label])
		},
		Color: color,
	})
}

// commandTable renders the commands by total time.
func commandTable(report *StatsReport, color bool) string {
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

// dim renders a line that is there to be read past.
func dim(text string, color bool) string {
	if !color {
		return text
	}
	return colors.Dim(text)
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

// ago renders how long ago a time was, and a dash for no time at all.
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
