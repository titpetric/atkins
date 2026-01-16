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

type GoroutineLifetime struct {
	ID        uint64
	Start     float64
	End       float64
	EventIDs  []string
	RunLabels []string
}

func main() {
	inputFile := flag.String("i", "atkins.log", "Input atkins log file (YAML)")
	outputFormat := flag.String("format", "text", "Output format: text, csv, mermaid")
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

	// Check if goroutine IDs are present
	hasGoroutineIDs := false
	for _, e := range log.Events {
		if e.GoroutineID > 0 {
			hasGoroutineIDs = true
			break
		}
	}

	if !hasGoroutineIDs {
		fmt.Fprintf(os.Stderr, "Warning: No goroutine IDs found in log. Run atkins with --debug to enable goroutine tracking.\n")
	}

	lifetimes := computeGoroutineLifetimes(log.Events)

	switch *outputFormat {
	case "text":
		outputText(lifetimes, log)
	case "csv":
		outputCSV(lifetimes)
	case "mermaid":
		outputMermaid(lifetimes, log)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *outputFormat)
		os.Exit(1)
	}
}

func computeGoroutineLifetimes(events []*Event) map[uint64]*GoroutineLifetime {
	lifetimes := make(map[uint64]*GoroutineLifetime)

	for _, e := range events {
		gid := e.GoroutineID
		if gid == 0 {
			continue
		}

		end := e.Start + e.Duration

		lt, ok := lifetimes[gid]
		if !ok {
			lt = &GoroutineLifetime{
				ID:    gid,
				Start: e.Start,
				End:   end,
			}
			lifetimes[gid] = lt
		}

		if e.Start < lt.Start {
			lt.Start = e.Start
		}
		if end > lt.End {
			lt.End = end
		}

		lt.EventIDs = append(lt.EventIDs, e.ID)
		lt.RunLabels = append(lt.RunLabels, truncate(e.Run, 40))
	}

	return lifetimes
}

func outputText(lifetimes map[uint64]*GoroutineLifetime, log Log) {
	sorted := sortLifetimes(lifetimes)

	fmt.Printf("Goroutine Timeline for: %s\n", log.Metadata.Pipeline)
	fmt.Printf("%-12s %-10s %-10s %-10s %s\n", "GoroutineID", "Start(s)", "End(s)", "Duration", "Events")
	fmt.Println(strings.Repeat("-", 80))

	for _, lt := range sorted {
		duration := lt.End - lt.Start
		eventCount := len(lt.EventIDs)
		fmt.Printf("%-12d %-10.3f %-10.3f %-10.3f %d events\n",
			lt.ID, lt.Start, lt.End, duration, eventCount)

		// Show first few runs for context
		for i, run := range lt.RunLabels {
			if i >= 3 {
				fmt.Printf("             ... and %d more\n", len(lt.RunLabels)-3)
				break
			}
			fmt.Printf("             - %s\n", run)
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("Total goroutines observed: %d\n", len(sorted))

	// Check for overlapping lifetimes (concurrent execution)
	overlaps := findOverlaps(sorted)
	if len(overlaps) > 0 {
		fmt.Printf("Concurrent execution periods: %d\n", len(overlaps))
		for _, o := range overlaps {
			fmt.Printf("  %.3fs - %.3fs: goroutines %v\n", o.Start, o.End, o.GoroutineIDs)
		}
	}
}

func outputCSV(lifetimes map[uint64]*GoroutineLifetime) {
	sorted := sortLifetimes(lifetimes)

	fmt.Println("goroutine_id,start,end,duration,event_count,first_event")
	for _, lt := range sorted {
		duration := lt.End - lt.Start
		firstEvent := ""
		if len(lt.EventIDs) > 0 {
			firstEvent = lt.EventIDs[0]
		}
		fmt.Printf("%d,%.6f,%.6f,%.6f,%d,%q\n",
			lt.ID, lt.Start, lt.End, duration, len(lt.EventIDs), firstEvent)
	}
}

func outputMermaid(lifetimes map[uint64]*GoroutineLifetime, log Log) {
	sorted := sortLifetimes(lifetimes)
	if len(sorted) == 0 {
		fmt.Fprintln(os.Stderr, "No goroutine data to render")
		return
	}

	fmt.Println("gantt")
	fmt.Printf("    title Goroutine Timeline: %s\n", sanitizeMermaid(log.Metadata.Pipeline))
	fmt.Println("    dateFormat X")
	fmt.Println("    axisFormat %s")

	// Scale to milliseconds for better readability
	for _, lt := range sorted {
		startMs := int64(lt.Start * 1000)
		durationMs := int64((lt.End - lt.Start) * 1000)
		if durationMs < 1 {
			durationMs = 1
		}

		label := fmt.Sprintf("G%d (%d events)", lt.ID, len(lt.EventIDs))
		fmt.Printf("    %s : %d, %dms\n", sanitizeMermaid(label), startMs, durationMs)
	}
}

type Overlap struct {
	Start        float64
	End          float64
	GoroutineIDs []uint64
}

func findOverlaps(lifetimes []*GoroutineLifetime) []Overlap {
	if len(lifetimes) < 2 {
		return nil
	}

	var overlaps []Overlap

	// Simple O(n²) check for overlaps
	for i := 0; i < len(lifetimes); i++ {
		for j := i + 1; j < len(lifetimes); j++ {
			a, b := lifetimes[i], lifetimes[j]

			// Check if they overlap
			overlapStart := max(a.Start, b.Start)
			overlapEnd := min(a.End, b.End)

			if overlapStart < overlapEnd {
				overlaps = append(overlaps, Overlap{
					Start:        overlapStart,
					End:          overlapEnd,
					GoroutineIDs: []uint64{a.ID, b.ID},
				})
			}
		}
	}

	return overlaps
}

func sortLifetimes(lifetimes map[uint64]*GoroutineLifetime) []*GoroutineLifetime {
	result := make([]*GoroutineLifetime, 0, len(lifetimes))
	for _, lt := range lifetimes {
		result = append(result, lt)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func sanitizeMermaid(s string) string {
	s = strings.ReplaceAll(s, ":", " -")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, ";", " ")
	return s
}
