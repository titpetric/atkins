package agent

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/titpetric/atkins/agent/view"
	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/runner"
)

// ExecutionProgressExported is exported for testing. Use newExecutionProgress in production.
type ExecutionProgressExported = executionProgress

// NewExecutionProgressForTest creates an executionProgress for testing.
func NewExecutionProgressForTest() *executionProgress {
	return newExecutionProgress()
}

// runningJob tracks a currently running job.
type runningJob struct {
	StartedAt time.Time
	Parents   []string
}

// executionProgress tracks job execution state for the TUI.
type executionProgress struct {
	Running map[string]*runningJob // job name -> running state
	Passed  int
	Failed  int
	Skipped int
}

func newExecutionProgress() *executionProgress {
	return &executionProgress{Running: make(map[string]*runningJob)}
}

// Apply updates state from a job progress event.
func (p *executionProgress) Apply(ev runner.JobProgressEvent) {
	switch ev.Status {
	case runner.JobProgressRunning:
		p.Running[ev.JobName] = &runningJob{
			StartedAt: ev.StartedAt,
			Parents:   ev.Parents,
		}
	case runner.JobProgressPassed:
		delete(p.Running, ev.JobName)
		p.Passed++
	case runner.JobProgressFailed:
		delete(p.Running, ev.JobName)
		p.Failed++
	case runner.JobProgressSkipped:
		delete(p.Running, ev.JobName)
		p.Skipped++
	}
}

// formatBreadcrumb renders a job name with its parent chain, styled with colors.
// e.g. parents=["default"], name="fmt" -> "default > fmt" with dim separators.
func formatBreadcrumb(parents []string, name string) string {
	if len(parents) == 0 {
		return colors.BrightWhite(name)
	}
	parts := make([]string, 0, len(parents)+1)
	for _, p := range parents {
		parts = append(parts, colors.BrightWhite(p))
	}
	parts = append(parts, colors.BrightWhite(name))
	return strings.Join(parts, colors.Dim(" > "))
}

// StatusLine renders a compact status line showing current progress.
func (p *executionProgress) StatusLine(now time.Time) string {
	if p == nil {
		return ""
	}

	var parts []string
	for name, rj := range p.Running {
		parts = append(parts, fmt.Sprintf("%s %s",
			formatBreadcrumb(rj.Parents, name),
			colors.Dim("("+view.FormatJobDuration(now.Sub(rj.StartedAt))+")")))
	}
	sort.Strings(parts)

	var b strings.Builder
	if len(parts) > 0 {
		b.WriteString("Running: ")
		b.WriteString(strings.Join(parts, ", "))
	}

	if p.Passed > 0 || p.Failed > 0 {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		if p.Failed > 0 {
			b.WriteString(colors.BrightRed(fmt.Sprintf("%d failed", p.Failed)))
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%d OK", p.Passed))
	}

	return b.String()
}
