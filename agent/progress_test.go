package agent_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/agent"
	"github.com/titpetric/atkins/runner"
)

func TestExecutionProgress_Apply(t *testing.T) {
	p := agent.NewExecutionProgressForTest()
	now := time.Now()

	p.Apply(runner.JobProgressEvent{
		JobName:   "build",
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})

	line := p.StatusLine(now.Add(2 * time.Second))
	assert.Contains(t, line, "build")
	assert.Contains(t, line, "2.00s")

	p.Apply(runner.JobProgressEvent{
		JobName:  "build",
		Status:   runner.JobProgressPassed,
		Duration: 2 * time.Second,
	})

	line = p.StatusLine(now)
	assert.Contains(t, line, "1 OK")
	assert.NotContains(t, line, "Running")
}

func TestExecutionProgress_FailedJobs(t *testing.T) {
	p := agent.NewExecutionProgressForTest()
	now := time.Now()

	p.Apply(runner.JobProgressEvent{
		JobName:   "test",
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})
	p.Apply(runner.JobProgressEvent{
		JobName:  "test",
		Status:   runner.JobProgressFailed,
		Duration: 1 * time.Second,
	})

	line := p.StatusLine(now)
	assert.Contains(t, line, "1 failed")
}

func TestExecutionProgress_MultipleRunning(t *testing.T) {
	p := agent.NewExecutionProgressForTest()
	now := time.Now()

	p.Apply(runner.JobProgressEvent{
		JobName:   "build",
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})
	p.Apply(runner.JobProgressEvent{
		JobName:   "lint",
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})

	line := p.StatusLine(now.Add(1 * time.Second))
	assert.Contains(t, line, "build")
	assert.Contains(t, line, "lint")
	assert.Contains(t, line, "Running:")
}

func TestExecutionProgress_Skipped(t *testing.T) {
	p := agent.NewExecutionProgressForTest()

	p.Apply(runner.JobProgressEvent{
		JobName: "deploy",
		Status:  runner.JobProgressSkipped,
	})

	line := p.StatusLine(time.Now())
	// Skipped jobs don't show up in the status line count
	assert.NotContains(t, line, "deploy")
}

func TestExecutionProgress_Nil(t *testing.T) {
	var p *agent.ExecutionProgressExported
	assert.Empty(t, p.StatusLine(time.Now()))
}

func TestExecutionProgress_NestedBreadcrumb(t *testing.T) {
	p := agent.NewExecutionProgressForTest()
	now := time.Now()

	// Parent job "default" starts
	p.Apply(runner.JobProgressEvent{
		JobName:   "default",
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})
	// Nested job "fmt" starts inside "default"
	p.Apply(runner.JobProgressEvent{
		JobName:   "fmt",
		Parents:   []string{"default"},
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})

	line := p.StatusLine(now.Add(1 * time.Second))
	assert.Contains(t, line, "default")
	assert.Contains(t, line, "fmt")
	assert.Contains(t, line, ">") // breadcrumb separator
}

func TestExecutionProgress_DeepNesting(t *testing.T) {
	p := agent.NewExecutionProgressForTest()
	now := time.Now()

	p.Apply(runner.JobProgressEvent{
		JobName:   "lint",
		Parents:   []string{"default", "check"},
		Status:    runner.JobProgressRunning,
		StartedAt: now,
	})

	line := p.StatusLine(now)
	assert.Contains(t, line, "default")
	assert.Contains(t, line, "check")
	assert.Contains(t, line, "lint")
}
