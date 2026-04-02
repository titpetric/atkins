package agent_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/titpetric/atkins/agent"
)

func TestJobView_NewJobView(t *testing.T) {
	v := agent.NewJobView()
	assert.NotNil(t, v)
}

func TestRenderJobEntry_Running(t *testing.T) {
	result := agent.RenderJobEntry("go:test", true, false, 0, "")
	assert.Contains(t, result, "go:test")
	assert.Contains(t, result, "●") // Running indicator
}

func TestRenderJobEntry_Passed(t *testing.T) {
	result := agent.RenderJobEntry("go:test", false, false, 500*time.Millisecond, "")
	assert.Contains(t, result, "go:test")
	assert.Contains(t, result, "✓") // Pass indicator
	assert.Contains(t, result, "500ms")
}

func TestRenderJobEntry_Failed(t *testing.T) {
	result := agent.RenderJobEntry("go:test", false, true, 1*time.Second, "assertion failed")
	assert.Contains(t, result, "go:test")
	assert.Contains(t, result, "✗") // Fail indicator
	assert.Contains(t, result, "1.00s")
	assert.Contains(t, result, "assertion failed")
}

func TestRenderJobEntry_FailedLongError(t *testing.T) {
	longError := strings.Repeat("x", 100)
	result := agent.RenderJobEntry("go:test", false, true, 1*time.Second, longError)

	// Error should be truncated to fit on one line
	// The truncation happens at 80 chars -> 77 + "..."
	assert.Contains(t, result, "...")
	// Check that the full error is not present
	assert.NotContains(t, result, longError)
}

func TestRenderJobSummary_AllPassed(t *testing.T) {
	result := agent.RenderJobSummary(5, 5, 0, 2*time.Second)
	assert.Contains(t, result, "DONE")
	assert.Contains(t, result, "5 jobs")
	assert.Contains(t, result, "2.00s")
}

func TestRenderJobSummary_SomeFailed(t *testing.T) {
	result := agent.RenderJobSummary(5, 3, 2, 3*time.Second)
	assert.Contains(t, result, "FAIL")
	assert.Contains(t, result, "5 jobs")
	assert.Contains(t, result, "2 failed")
}

func TestFormatJobDuration_Milliseconds(t *testing.T) {
	tests := []struct {
		d        time.Duration
		contains string
	}{
		{500 * time.Microsecond, "<1ms"},
		{50 * time.Millisecond, "50ms"},
		{999 * time.Millisecond, "999ms"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := agent.RenderJobEntry("test", false, false, tt.d, "")
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFormatJobDuration_Seconds(t *testing.T) {
	tests := []struct {
		d        time.Duration
		contains string
	}{
		{1 * time.Second, "1.00s"},
		{1500 * time.Millisecond, "1.50s"},
		{59 * time.Second, "59.00s"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := agent.RenderJobEntry("test", false, false, tt.d, "")
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFormatJobDuration_Minutes(t *testing.T) {
	tests := []struct {
		d        time.Duration
		contains string
	}{
		{1 * time.Minute, "1m00s"},
		{90 * time.Second, "1m30s"},
		{5*time.Minute + 30*time.Second, "5m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := agent.RenderJobEntry("test", false, false, tt.d, "")
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestFormatJobDuration_Hours(t *testing.T) {
	tests := []struct {
		d        time.Duration
		contains string
	}{
		{1 * time.Hour, "1h00m"},
		{90 * time.Minute, "1h30m"},
		{2*time.Hour + 15*time.Minute, "2h15m"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := agent.RenderJobEntry("test", false, false, tt.d, "")
			assert.Contains(t, result, tt.contains)
		})
	}
}
