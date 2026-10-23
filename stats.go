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

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/config"
	"github.com/titpetric/atkins/markdown"
	"github.com/titpetric/atkins/storage"
)

// statsLimit is how many commands the report lists. Twenty fits a
// terminal, and the tail of a long tail is not what anyone is looking
// for.
const statsLimit = 20

// runStats reports the recorded commands by the time they have taken,
// the most expensive first.
//
// The question it answers is where the time goes: not which command is
// slowest, but which one costs the most once its frequency is counted. A
// test run of four seconds that happens two hundred times a day outranks
// a release build of three minutes that happens weekly, and the ordering
// by total time says so.
func runStats(ctx context.Context, settings *config.Config, opts *Options) error {
	store, err := openStorage(settings)
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}
	defer store.Close()

	stats, err := store.Stats(ctx, storage.StatsRequest{Limit: statsLimit})
	if err != nil {
		return fmt.Errorf("%s %v", colors.BrightRed("ERROR:"), err)
	}

	switch {
	case opts.JSON:
		return writeJSON(os.Stdout, stats)
	case opts.YAML:
		return writeYAML(os.Stdout, stats)
	}

	writeStats(os.Stdout, store.Path(), stats, isTerminal(os.Stdout))
	return nil
}

// statsReport is one command of the report as a program reads it.
//
// The durations are milliseconds, as the database keeps them, because a
// reader that charts them wants a number and not a word.
type statsReport struct {
	Command   string     `json:"command" yaml:"command"`
	Tool      string     `json:"tool" yaml:"tool"`
	Runs      int64      `json:"runs" yaml:"runs"`
	TotalMs   int64      `json:"total_ms" yaml:"total_ms"`
	AverageMs int64      `json:"average_ms" yaml:"average_ms"`
	MaxMs     int64      `json:"max_ms" yaml:"max_ms"`
	Failures  int64      `json:"failures" yaml:"failures"`
	LastRunAt *time.Time `json:"last_run_at,omitempty" yaml:"last_run_at,omitempty"`
}

// reports maps the rows onto what the machine-readable output carries.
func reports(stats []storage.CommandStat) []statsReport {
	out := make([]statsReport, 0, len(stats))
	for _, stat := range stats {
		out = append(out, statsReport{
			Command:   stat.Normalized,
			Tool:      stat.Tool,
			Runs:      stat.Runs,
			TotalMs:   stat.TotalMs,
			AverageMs: stat.AvgMs,
			MaxMs:     stat.MaxMs,
			Failures:  stat.Failures,
			LastRunAt: stat.LastAt,
		})
	}
	return out
}

// writeJSON emits the report as JSON, for a program reading it.
func writeJSON(w io.Writer, stats []storage.CommandStat) error {
	data, err := json.MarshalIndent(reports(stats), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// writeYAML emits the report as YAML.
func writeYAML(w io.Writer, stats []storage.CommandStat) error {
	data, err := yaml.Marshal(reports(stats))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(data))
	return err
}

// writeStats renders the report for a person.
//
// On a terminal the table is drawn with borders and colour, the way
// `atkins --help` draws its tables. Redirected to a file or a pipe it is
// plain markdown, so the same command serves a terminal and a document
// without a flag to say which.
func writeStats(w io.Writer, path string, stats []storage.CommandStat, color bool) {
	if len(stats) == 0 {
		fmt.Fprintf(w, "%s nothing recorded yet in %s\n", colors.BrightYellow("INFO:"), path)
		return
	}

	rows := []string{
		"| Command | Runs | Total | Avg | Max | Failed | Last |",
		"| --- | --- | --- | --- | --- | --- | --- |",
	}
	for _, stat := range stats {
		rows = append(rows, "| "+strings.Join([]string{
			stat.Normalized,
			strconv.FormatInt(stat.Runs, 10),
			duration(stat.TotalMs),
			duration(stat.AvgMs),
			duration(stat.MaxMs),
			strconv.FormatInt(stat.Failures, 10),
			ago(stat.LastAt),
		}, " | ")+" |")
	}

	table := strings.Join(rows, "\n") + "\n"
	if !color {
		fmt.Fprintf(w, "%s\n\n%s", path, table)
		return
	}

	fmt.Fprintf(w, "%s\n\n%s\n", colors.BrightWhite(path), markdown.Render(table, true))
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
