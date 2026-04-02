package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/titpetric/atkins/colors"
)

// JobStatus represents the execution status of a job.
type JobStatus int

// JobStatus constants.
const (
	JobStatusPending JobStatus = iota
	JobStatusRunning
	JobStatusPassed
	JobStatusFailed
	JobStatusSkipped
)

// JobEntry tracks a single job's execution progress.
type JobEntry struct {
	Name      string
	Status    JobStatus
	StartTime time.Time
	Duration  time.Duration
	Error     string
	Steps     []StepEntry
}

// StepEntry tracks a single step within a job.
type StepEntry struct {
	Name     string
	Status   JobStatus
	Duration time.Duration
	Error    string
}

// JobView renders job execution in gotestsum-style format.
type JobView struct {
	entries []JobEntry
}

// NewJobView creates a new job view.
func NewJobView() *JobView {
	return &JobView{}
}

// StartJob begins tracking a new job.
func (v *JobView) StartJob(name string) {
	v.entries = append(v.entries, JobEntry{
		Name:      name,
		Status:    JobStatusRunning,
		StartTime: time.Now(),
	})
}

// EndJob marks a job as complete.
func (v *JobView) EndJob(name string, success bool, errMsg string) {
	for i := range v.entries {
		if v.entries[i].Name == name && v.entries[i].Status == JobStatusRunning {
			v.entries[i].Duration = time.Since(v.entries[i].StartTime)
			if success {
				v.entries[i].Status = JobStatusPassed
			} else {
				v.entries[i].Status = JobStatusFailed
				v.entries[i].Error = errMsg
			}
			break
		}
	}
}

// RenderEntry renders a single job entry in gotestsum style.
// Format: ✓ job:name (0.12s)
//
//	or: ✗ job:name (0.12s)
//	        → Error: <message>
func RenderJobEntry(name string, running bool, failed bool, duration time.Duration, errMsg string) string {
	var lines []string

	durStr := formatJobDuration(duration)

	if running {
		lines = append(lines, fmt.Sprintf("  %s %s",
			colors.BrightYellow("●"),
			colors.BrightWhite(name)))
	} else if failed {
		lines = append(lines, fmt.Sprintf("  %s %s %s",
			colors.BrightRed("✗"),
			colors.BrightWhite(name),
			colors.Dim("("+durStr+")")))
		if errMsg != "" {
			// Show only the first line of error for compact display
			firstLine := strings.SplitN(errMsg, "\n", 2)[0]
			if len(firstLine) > 80 {
				firstLine = firstLine[:77] + "..."
			}
			lines = append(lines, fmt.Sprintf("      %s %s",
				colors.BrightRed("→"),
				colors.Dim(firstLine)))
		}
	} else {
		lines = append(lines, fmt.Sprintf("  %s %s %s",
			colors.BrightGreen("✓"),
			colors.BrightWhite(name),
			colors.Dim("("+durStr+")")))
	}

	return strings.Join(lines, "\n")
}

// RenderSummary renders a summary line in gotestsum style.
// Format: DONE 5 jobs, 1 failure in 2.34s
func RenderJobSummary(total, passed, failed int, duration time.Duration) string {
	durStr := formatJobDuration(duration)

	if failed == 0 {
		return fmt.Sprintf("%s %d jobs in %s",
			colors.BrightGreen("DONE"),
			total,
			durStr)
	}

	return fmt.Sprintf("%s %d jobs, %s in %s",
		colors.BrightRed("FAIL"),
		total,
		colors.BrightRed(fmt.Sprintf("%d failed", failed)),
		durStr)
}

// formatJobDuration formats duration for display (similar to gotestsum).
func formatJobDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", mins, secs)
	}
	hrs := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", hrs, mins)
}
