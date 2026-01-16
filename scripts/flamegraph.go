//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type Result string

const (
	ResultPass    Result = "pass"
	ResultFail    Result = "fail"
	ResultSkipped Result = "skipped"
)

type Event struct {
	ID          string  `yaml:"id"`
	Run         string  `yaml:"run"`
	Result      Result  `yaml:"result"`
	Start       float64 `yaml:"start"`
	Duration    float64 `yaml:"duration"`
	Error       string  `yaml:"error,omitempty"`
	GoroutineID uint64  `yaml:"goroutine_id,omitempty"`
}

type RunMetadata struct {
	RunID      string `yaml:"run_id"`
	Pipeline   string `yaml:"pipeline,omitempty"`
	File       string `yaml:"file,omitempty"`
	ModulePath string `yaml:"module_path,omitempty"`
}

type RunSummary struct {
	Duration     float64 `yaml:"duration"`
	TotalSteps   int     `yaml:"total_steps"`
	PassedSteps  int     `yaml:"passed_steps"`
	FailedSteps  int     `yaml:"failed_steps"`
	SkippedSteps int     `yaml:"skipped_steps"`
	Result       Result  `yaml:"result"`
}

type Log struct {
	Metadata RunMetadata `yaml:"metadata"`
	Events   []*Event    `yaml:"events"`
	Summary  *RunSummary `yaml:"summary,omitempty"`
}

func main() {
	inputFile := flag.String("i", "atkins.log", "Input atkins log file (YAML)")
	outputFormat := flag.String("format", "folded", "Output format: folded (for flamegraph.pl)")
	unitMs := flag.Bool("ms", false, "Use milliseconds instead of microseconds for duration")
	flag.Parse()

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var log Log
	if err := yaml.Unmarshal(data, &log); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	switch *outputFormat {
	case "folded":
		outputFolded(log, *unitMs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *outputFormat)
		os.Exit(1)
	}
}

// outputFolded outputs events in folded stack format for flamegraph.pl
// Format: stack;frame;frame value
// The stack is built from the event ID hierarchy
func outputFolded(log Log, unitMs bool) {
	// Sort events by start time
	events := make([]*Event, len(log.Events))
	copy(events, log.Events)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start < events[j].Start
	})

	// Build folded stacks
	// Event ID format: jobs.<jobName> or jobs.<jobName>.steps.<N>
	// We want: pipeline;job;step value

	pipeline := log.Metadata.Pipeline
	if pipeline == "" {
		pipeline = "atkins"
	}
	// Sanitize pipeline name (remove spaces, special chars)
	pipeline = sanitize(pipeline)

	for _, event := range events {
		stack := buildStack(pipeline, event.ID, event.Run)

		// Duration in microseconds (default) or milliseconds
		var value int64
		if unitMs {
			value = int64(event.Duration * 1000) // ms
		} else {
			value = int64(event.Duration * 1000000) // μs
		}

		// Skip zero-duration events
		if value <= 0 {
			continue
		}

		fmt.Printf("%s %d\n", stack, value)
	}
}

// buildStack creates a semicolon-separated stack from event ID and run command
func buildStack(pipeline, id, run string) string {
	parts := []string{pipeline}

	// Parse ID: jobs.<jobName> or jobs.<jobName>.steps.<N>
	if strings.HasPrefix(id, "jobs.") {
		remainder := strings.TrimPrefix(id, "jobs.")

		// Check if it's a step (contains .steps.)
		if idx := strings.Index(remainder, ".steps."); idx > 0 {
			jobName := remainder[:idx]
			parts = append(parts, sanitize(jobName))
			// Add the run command as the leaf
			parts = append(parts, sanitize(run))
		} else {
			// It's a job-level event
			parts = append(parts, sanitize(remainder))
		}
	}

	return strings.Join(parts, ";")
}

// sanitize removes/replaces characters that might cause issues in flamegraph
func sanitize(s string) string {
	// Replace semicolons (stack separator)
	s = strings.ReplaceAll(s, ";", ":")
	// Replace spaces
	s = strings.ReplaceAll(s, " ", "_")
	// Replace newlines
	s = strings.ReplaceAll(s, "\n", "_")
	return s
}
